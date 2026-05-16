package federation

import (
	"crypto/ed25519"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/caio-bernardo/dragonite/internal/types"
	"github.com/caio-bernardo/dragonite/internal/util"
)

type Handler struct {
	config *types.ServerConfig
}

func NewHandler(config *types.ServerConfig) *Handler {
	return &Handler{config: config}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /_matrix/federation/v1/version", h.getVersion)
	mux.HandleFunc("GET /_matrix/key/v2/server", h.getServerKey)
}

func (h *Handler) getVersion(w http.ResponseWriter, r *http.Request) {
	res := VersionResponse{}
	res.Server.Name = h.config.ServerName
	res.Server.Version = h.config.Version
	util.WriteJSON(w, http.StatusOK, res)
}

func (h *Handler) getServerKey(w http.ResponseWriter, r *http.Request) {
	resp := ServerKeyResponse{}

	resp.ServerName = h.config.ServerName
	// Validade de 1 ano
	resp.ValidUntilTS = time.Now().Add(365 * 24 * time.Hour).UnixMilli()
	publicKey := base64.RawStdEncoding.EncodeToString(h.config.PublicKey)
	resp.VerifyKeys = map[string]VerifyKey{
		h.config.KeyID: {
			Key: publicKey,
		},
	}

	// Criptografia
	canonicalJson, err := util.CanonicalJSON(resp)
	if err != nil {
		util.WriteError(w, http.StatusInternalServerError, types.NewErrorResponse(types.M_BAD_JSON, err.Error()))
		return
	}
	signatureBytes := ed25519.Sign(h.config.PrivateKey, canonicalJson)
	signatureBase64 := base64.RawStdEncoding.EncodeToString(signatureBytes)

	// add signature
	resp.Signatures = map[string]map[string]string{
		h.config.ServerName: {
			h.config.KeyID: signatureBase64,
		},
	}

	util.WriteJSON(w, http.StatusOK, resp)
}
