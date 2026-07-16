package roomkeys

import (
	"errors"
	"log"
	"net/http"

	"github.com/caio-bernardo/dragonite/internal/delivery/http_adapter/httputil"
	"github.com/caio-bernardo/dragonite/internal/domain/types"
	"github.com/caio-bernardo/dragonite/internal/usecase"
)

// Handler agrupa as rotas de backup de chaves E2EE (room_keys) do cliente Matrix
type Handler struct {
	backupService *usecase.BackupService
}

// NewHandler cria um Handler de room_keys com o serviço injetado
func NewHandler(backupService *usecase.BackupService) *Handler {
	return &Handler{backupService: backupService}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMiddleware httputil.Middleware) {
	// TODO: adicionar rate limiting por userID antes do authMiddleware quando
	// a infraestrutura de rate limiting for implementada no projeto.
	mux.Handle("GET /_matrix/client/v3/room_keys/version", authMiddleware(http.HandlerFunc(h.getLatestVersion)))
	mux.Handle("POST /_matrix/client/v3/room_keys/version", authMiddleware(http.HandlerFunc(h.createVersion)))
}

// getLatestVersion retorna informações sobre a versão mais recente do backup de chaves
// GET /_matrix/client/v3/room_keys/version
func (h *Handler) getLatestVersion(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(types.UserIDKey).(string)
	if !ok || userID == "" {
		httputil.WriteMatrixError(w, http.StatusUnauthorized, httputil.M_MISSING_TOKEN, "Missing or invalid auth token")
		return
	}

	backup, err := h.backupService.GetLatestBackupVersion(r.Context(), userID)
	if err != nil {
		if errors.Is(err, usecase.ErrBackupNotFound) {
			httputil.WriteMatrixError(w, http.StatusNotFound, httputil.M_NOT_FOUND, "No current backup version")
			return
		}

		log.Printf("[ERROR] GET /_matrix/client/v3/room_keys/version (user=%s): %v", userID, err)
		httputil.WriteMatrixError(w, http.StatusInternalServerError, httputil.M_UNKNOWN, "Failed to fetch backup version")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, BackupVersionResponse{
		Algorithm: backup.Algorithm,
		AuthData:  backup.AuthData,
		Count:     backup.Count,
		ETag:      backup.ETag,
		Version:   backup.VersionString(),
	})
}

// createVersion cria uma nova versão de backup de chaves
// POST /_matrix/client/v3/room_keys/version
func (h *Handler) createVersion(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(types.UserIDKey).(string)
	if !ok || userID == "" {
		httputil.WriteMatrixError(w, http.StatusUnauthorized, httputil.M_MISSING_TOKEN, "Missing or invalid auth token")
		return
	}

	var req CreateBackupVersionRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.WriteMatrixError(w, http.StatusBadRequest, httputil.M_NOT_JSON, "Request did not contain valid JSON")
		return
	}

	if req.Algorithm == "" || len(req.AuthData) == 0 {
		httputil.WriteMatrixError(w, http.StatusBadRequest, httputil.M_MISSING_PARAM, "Missing algorithm or auth_data")
		return
	}

	backup, err := h.backupService.CreateBackupVersion(r.Context(), usecase.CreateBackupParams{
		UserID:    userID,
		Algorithm: req.Algorithm,
		AuthData:  req.AuthData,
	})
	if err != nil {
		log.Printf("[ERROR] POST /_matrix/client/v3/room_keys/version (user=%s): %v", userID, err)
		httputil.WriteMatrixError(w, http.StatusInternalServerError, httputil.M_UNKNOWN, "Failed to create backup version")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, CreateBackupVersionResponse{
		Version: backup.VersionString(),
	})
}