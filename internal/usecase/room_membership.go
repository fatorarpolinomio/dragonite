package usecase

import (
	"context"
	"fmt"

	"github.com/caio-bernardo/dragonite/internal/util"
)

type RoomMembershipService struct {
	uow              WorkUnit
	authRuleResolver *AuthRuleResolver
	canalRepo        CanalStorage
	eventRepo        EventoStorage
}

func NewRoomMembershipService(uow WorkUnit, canalRepo CanalStorage, eventRepo EventoStorage, authRuleResolver *AuthRuleResolver) *RoomMembershipService {
	return &RoomMembershipService{
		uow:              uow,
		authRuleResolver: authRuleResolver,
		canalRepo:        canalRepo,
		eventRepo:        eventRepo,
	}
}

func (s *RoomMembershipService) LeaveRoom(ctx context.Context, userID, roomID string) error {
	// 1. Verify they are actually in the room (or invited)
	currentStatus, err := s.canalRepo.GetUserMembership(ctx, roomID, userID)
	if err != nil || (currentStatus != "join" && currentStatus != "invite") {
		return fmt.Errorf("user is not in a state to leave this room")
	}

	// 2. Build the Leave Event
	leaveEvent := buildLeaveEvent(roomID, userID)

	// 3. Resolve DAG dependencies (Use the method we built earlier)
	prevs, auths, err := s.authRuleResolver.ResolveEventDependencies(ctx, roomID, userID, "m.room.member", &userID)
	if err != nil {
		return err
	}
	leaveEvent.PrevEventos = prevs
	leaveEvent.AuthEventos = auths
	maxDepth, err := s.eventRepo.GetMaxDepthFromEventos(ctx, prevs)
	if err != nil {
		return fmt.Errorf("failed to get event depth: %w", err)
	}
	leaveEvent.Depth = maxDepth + 1
	// 4. Hash it
	eventID, _ := util.HashMatrixEvent(leaveEvent)
	leaveEvent.ID = eventID

	// 5. Transactional Save
	err = s.uow.Execute(ctx, func(txCtx context.Context) error {
		// A. Save the event to the DAG
		if err := s.eventRepo.SaveEvento(txCtx, leaveEvent); err != nil {
			return err
		}
		// Update DAG Extremities
		if err := s.canalRepo.UpdateForwardExtremities(txCtx, roomID, []string{eventID}); err != nil {
			return err
		}

		// Update the Room's Current State for this state_key
		if err := s.canalRepo.UpsertCurrentState(txCtx, roomID, "m.room.member", userID, eventID); err != nil {
			return err
		}

		// B. Update the denormalized fast-lookup table
		if err := s.canalRepo.UpsertMembership(txCtx, roomID, userID, "leave"); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 6. Notify local clients (so the room disappears from their UI instantly)
	// NOTE: client are automatically notified after transaction is completed by notifier package
	return nil
}

func (s *RoomMembershipService) JoinLocalRoom(ctx context.Context, userID, roomID string) error {

	// 1. Fetch the Join Rules
	joinRule, err := s.canalRepo.GetJoinRule(ctx, roomID)
	if err != nil {
		return fmt.Errorf("room not found or missing join rules")
	}

	// 2. Validate Access
	if joinRule == "invite" {
		// They MUST have a pending invite in our database
		status, _ := s.canalRepo.GetUserMembership(ctx, roomID, userID)
		if status != "invite" {
			return fmt.Errorf("M_FORBIDDEN: You must be invited to join this room")
		}
	} else if joinRule != "public" {
		return fmt.Errorf("M_FORBIDDEN: Room is not public")
	}

	// 3. Build the Join Event

	joinEvent := buildJoinEvent(roomID, userID)

	// 4. Resolve DAG Dependencies
	prevs, auths, err := s.authRuleResolver.ResolveEventDependencies(ctx, roomID, userID, "m.room.member", &userID)
	if err != nil {
		return err
	}
	joinEvent.PrevEventos = prevs
	joinEvent.AuthEventos = auths
	maxDepth, err := s.eventRepo.GetMaxDepthFromEventos(ctx, prevs)
	if err != nil {
		return fmt.Errorf("failed to get event depth: %w", err)
	}
	joinEvent.Depth = maxDepth + 1
	// 5. Hash it
	eventID, _ := util.HashMatrixEvent(joinEvent)
	joinEvent.ID = eventID

	// 6. Transactional Save
	err = s.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := s.eventRepo.SaveEvento(txCtx, joinEvent); err != nil {
			return err
		}

		// Update DAG Extremities
		if err := s.canalRepo.UpdateForwardExtremities(txCtx, roomID, []string{eventID}); err != nil {
			return err
		}

		// Update the Room's Current State for this state_key
		if err := s.canalRepo.UpsertCurrentState(txCtx, roomID, "m.room.member", userID, eventID); err != nil {
			return err
		}

		// Update their status from "invite" (or null) to "join"
		if err := s.canalRepo.UpsertMembership(txCtx, roomID, userID, "join"); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	// 7. Notify clients
	// NOTE: client are automatically notified after transaction is completed by notifier package
	return nil
}
