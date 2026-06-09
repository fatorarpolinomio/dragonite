package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/caio-bernardo/dragonite/internal/domain"
	"github.com/caio-bernardo/dragonite/internal/domain/types"
	"github.com/caio-bernardo/dragonite/internal/util"
)

type RoomInteractionService struct {
	canalRepo        CanalStorage
	eventoRepo       EventoStorage
	fedService       FederationService
	authRuleResolver *AuthRuleResolver
	uow              WorkUnit
}

func NewRoomInteractionService(canalRepo CanalStorage, eventoRepo EventoStorage, fedService FederationService, authRuleResolver *AuthRuleResolver, uow WorkUnit) *RoomInteractionService {
	return &RoomInteractionService{
		canalRepo:        canalRepo,
		eventoRepo:       eventoRepo,
		fedService:       fedService,
		authRuleResolver: authRuleResolver,
		uow:              uow,
	}
}

type EventParams struct {
	RoomID    string
	SenderID  string
	Content   map[string]any
	EventType string
}

type StateParams struct {
	RoomID    string
	UserID    string
	EventType string
	StateKey  string
	Content   map[string]any
}

func (s *RoomInteractionService) SendStateEvent(ctx context.Context, params StateParams) (string, error) {
	// 1. Authorization: Check if the user is joined to the room
	status, err := s.canalRepo.GetUserMembership(ctx, params.RoomID, params.UserID)
	if err != nil || status != "join" {
		return "", types.ErrForbidden
	}
	// TODO: check powerlevel and if statekey starts with @ matches sender

	// 2. Build the Base State Event
	contentBytes, err := json.Marshal(params.Content)
	if err != nil {
		return "", fmt.Errorf("failed to marshal content: %w", err)
	}

	newEvent := &domain.Evento{
		CanalID:          params.RoomID,
		Sender:           params.UserID,
		Tipo:             params.EventType,
		StateKey:         &params.StateKey, // STATE events MUST have a state key (even if it's "")
		Content:          contentBytes,
		OrigemServidorTS: time.Now().UnixMilli(),
	}

	// 3. Resolve DAG Dependencies (The Timeline and the VIP Pass)
	prevs, auths, err := s.authRuleResolver.ResolveEventDependencies(ctx, params.RoomID, params.UserID, params.EventType, &params.StateKey)
	if err != nil {
		return "", fmt.Errorf("failed to resolve DAG dependencies: %w", err)
	}
	newEvent.PrevEventos = prevs
	newEvent.AuthEventos = auths

	maxDepth, err := s.eventoRepo.GetMaxDepthFromEventos(ctx, prevs)
	if err != nil {
		return "", fmt.Errorf("failed to get event depth: %w", err)
	}
	newEvent.Depth = maxDepth + 1

	// 4. Cryptographic Hashing
	eventID, err := util.HashMatrixEvent(newEvent)
	if err != nil {
		return "", fmt.Errorf("failed to hash event: %w", err)
	}
	newEvent.ID = eventID

	// 5. ATOMIC DATABASE TRANSACTION (The 3-Step State Update)
	err = s.uow.Execute(ctx, func(txCtx context.Context) error {
		// A. Save the historical event payload to the DAG
		if err := s.eventoRepo.SaveEvento(txCtx, newEvent); err != nil {
			return err
		}

		// B. Update the DAG Extremities (Move the timeline forward)
		if err := s.canalRepo.UpdateForwardExtremities(txCtx, params.RoomID, []string{eventID}); err != nil {
			return err
		}

		// C. Upsert the Current State (Overwrite the old state for this Type + StateKey)
		if err := s.canalRepo.UpsertCurrentState(txCtx, params.RoomID, params.EventType, params.StateKey, eventID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("transaction failed: %w", err)
	}

	// 6. Post-Transaction Side Effects
	// NOTE: Wake up local users listening on /sync so their UI updates instantly
	// Postgres handles notification on room level

	// Queue the state change to be pushed to remote servers
	_ = s.fedService.QueueOutgoing(ctx, *newEvent)

	return eventID, nil
}

func (s *RoomInteractionService) SendEvent(ctx context.Context, params EventParams) (string, error) {
	// 1. Authorization: Check if the user is joined to the room
	status, err := s.canalRepo.GetUserMembership(ctx, params.RoomID, params.SenderID)
	if err != nil || status != "join" {
		return "", types.ErrForbidden
	}

	// 2. Build the Base Event
	contentBytes, err := json.Marshal(params.Content)
	if err != nil {
		return "", fmt.Errorf("failed to marshal content: %w", err)
	}

	newEvent := &domain.Evento{
		CanalID:          params.RoomID,
		Sender:           params.SenderID,
		Tipo:             params.EventType,
		StateKey:         nil, // REGULAR events strictly have NO state key
		Content:          contentBytes,
		OrigemServidorTS: time.Now().UnixMilli(),
	}

	// 3. Resolve DAG Dependencies (The VIP Pass and the Timeline)
	prevs, auths, err := s.authRuleResolver.ResolveEventDependencies(ctx, params.RoomID, params.SenderID, params.EventType, nil)
	if err != nil {
		return "", fmt.Errorf("failed to resolve DAG dependencies: %w", err)
	}
	newEvent.PrevEventos = prevs
	newEvent.AuthEventos = auths

	maxDepth, err := s.eventoRepo.GetMaxDepthFromEventos(ctx, prevs)
	if err != nil {
		return "", fmt.Errorf("failed to get event depth: %w", err)
	}
	newEvent.Depth = maxDepth + 1

	// 4. Cryptographic Hashing
	eventID, err := util.HashMatrixEvent(newEvent)
	if err != nil {
		return "", fmt.Errorf("failed to hash event: %w", err)
	}
	newEvent.ID = eventID

	// 5. ATOMIC DATABASE TRANSACTION
	err = s.uow.Execute(ctx, func(txCtx context.Context) error {
		// A. Save the event payload
		if err := s.eventoRepo.SaveEvento(txCtx, newEvent); err != nil {
			return err
		}

		// B. Update the DAG Extremities
		// This query deletes the old extremities (prevs) and inserts the new EventID
		if err := s.canalRepo.UpdateForwardExtremities(txCtx, params.RoomID, []string{eventID}); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("transaction failed: %w", err)
	}

	// 6. Post-Transaction Side Effects (Waking up the network)
	// Wake up local users listening on /sync
	// NOTE: postgres handles notification on room level

	// Queue the event to be pushed to remote servers
	_ = s.fedService.QueueOutgoing(ctx, *newEvent)

	return eventID, nil
}
