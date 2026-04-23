package client

import (
	"net/http"
	"os"

	"github.com/caio-bernardo/dragonite/internal/repository"
	"github.com/caio-bernardo/dragonite/internal/services/client/auth"
	"github.com/caio-bernardo/dragonite/internal/services/client/rooms"
	"github.com/caio-bernardo/dragonite/internal/types"
	"github.com/caio-bernardo/dragonite/internal/util"
)

type Handler struct {
	userStore   repository.UserStore
	deviceStore repository.DeviceStore
	canalStore        repository.ChannelStore       
	usuarioCanalStore repository.UsuarioCanalStore
}

func NewHandler(userStore repository.UserStore, deviceStore repository.DeviceStore, canalStore repository.ChannelStore, usuarioCanalStore repository.UsuarioCanalStore) *Handler {
	return &Handler{userStore: userStore, deviceStore: deviceStore, canalStore: canalStore, usuarioCanalStore: usuarioCanalStore}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMiddleware types.Middleware) {

	auth := auth.NewHandler(h.userStore, h.deviceStore)
	roomHandler := rooms.NewHandler(h.canalStore, h.usuarioCanalStore, os.Getenv("SERVER_NAME"))

	mux.HandleFunc("GET /_matrix/client/versions", h.getVersions)

	// autenticação
	auth.RegisterRoutes(mux, authMiddleware)

	// sincronização de dados
	mux.HandleFunc("GET /_matrix/client/sync", util.UnimplementedHandler) // WARN: esse é o dificil

	// chats e manipulação de salas
	roomHandler.RegisterRoutes(mux, authMiddleware)

	// troca de eventos
	mux.HandleFunc("PUT /_matrix/client/v3/rooms/{roomId}/send/{eventType}/{txnId}", util.UnimplementedHandler)
	mux.HandleFunc("PUT /_matrix/client/v3/rooms/{roomId}/state/{eventType}/{stateKey}", util.UnimplementedHandler)
}

func (h *Handler) getVersions(w http.ResponseWriter, r *http.Request) {
	response := SupportedVersionsResponse{
		Versions: []string{"r0.0.5", "v1.18"},
	}
	util.WriteJSON(w, 200, response)
}
