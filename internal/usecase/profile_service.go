package usecase

import (
	"context"
	"log"

	"github.com/caio-bernardo/dragonite/internal/domain"
	"github.com/caio-bernardo/dragonite/internal/domain/types"
)

type ProfileService struct {
	userStore              UsuarioStorage
	canalStore             CanalStorage
	roomMembershipService  *RoomMembershipService  // <-- Campo novo
	roomInteractionService *RoomInteractionService // <-- Campo novo
}

func NewProfileService(userStore UsuarioStorage, canalStore CanalStorage, roomMembershipService *RoomMembershipService, roomInteractionService *RoomInteractionService) *ProfileService {
	return &ProfileService{
		userStore:              userStore,
		canalStore:             canalStore,
		roomMembershipService:  roomMembershipService,
		roomInteractionService: roomInteractionService,
	}
}

func (p *ProfileService) GetProfileByUserID(ctx context.Context, user_id string) (*domain.Profile, error) {
	if user_id == "" {
		return nil, types.ErrInvalidUserID
	}

	profile, err := p.userStore.GetProfileByID(ctx, user_id)
	if err != nil {
		return nil, types.ErrNotFound
	}
	if profile == nil {
		return nil, types.ErrNotFound
	}
	return profile, nil

}

type ProfileParams struct {
	DisplayName *string `json:"displayname,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

func (p *ProfileService) UpdateProfile(ctx context.Context, userID string, props ProfileParams) error {
	if userID == "" {
		return types.ErrInvalidUserID
	}

	profile := domain.Profile{
		IDUsuario:   userID,
		DisplayName: props.DisplayName,
		AvatarURL:   props.AvatarURL,
	}

	err := p.userStore.UpdateProfile(ctx, profile)
	if err != nil {
		return err
	}

	perfilCompleto, err := p.GetProfileByUserID(ctx, userID)
	if err != nil {
    return err
	}

	salas, err := p.roomMembershipService.GetJoinedRooms(ctx, userID) //[cite: 13]
	if err != nil {
    return err
	}

	for _, roomID := range salas {
    // Constrói o conteúdo exigido pelo Element para atualizar a interface
    content := map[string]any{
        "membership":  "join",
        "displayname": perfilCompleto.DisplayName,
        "avatar_url":  perfilCompleto.AvatarURL,
    }

    // Preenche a struct exata que o seu SendStateEvent exige
    params := StateParams{
        RoomID:    roomID,
        UserID:    userID,
        EventType: "m.room.member",
        StateKey:  userID, // O StateKey de um evento de membro deve ser o próprio userID
        Content:   content,
    }

    // Dispara o evento! Essa função já lida com a transação no banco e com a fila de federação
    _, err = p.roomInteractionService.SendStateEvent(ctx, params) //[cite: 12]

    if err != nil {
        // Apenas registre o erro no log, mas não interrompa o loop para não quebrar a sincronização das outras salas
        log.Printf("Erro ao propagar perfil na sala %s: %v", roomID, err)
    }
	}
	return nil
}
