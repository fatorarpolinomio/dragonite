package rooms

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/caio-bernardo/dragonite/internal/model"
	"github.com/caio-bernardo/dragonite/internal/repository"
	"github.com/caio-bernardo/dragonite/internal/types"
	"github.com/caio-bernardo/dragonite/internal/util"
)

// Handler agrupa as dependências dos handlers de rooms.
// Mesmo padrão de auth.Handler.
type Handler struct {
	canalStore       repository.ChannelStore      
	usuarioCanalStore repository.UsuarioCanalStore 
	serverName string 
}

func NewHandler(canalStore repository.ChannelStore, usuarioCanalStore repository.UsuarioCanalStore, serverName string) *Handler {
	return &Handler{canalStore: canalStore, usuarioCanalStore: usuarioCanalStore, serverName: serverName}
}

// RegisterRoutes registra todas as rotas de rooms no mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMiddleware types.Middleware) {
	// Não requer autenticação (spec permite listagem pública sem token)
	mux.HandleFunc("GET /_matrix/client/v3/publicRooms", h.getPublicRooms)

	// Requerem autenticação 
	mux.Handle("POST /_matrix/client/v3/createRoom", authMiddleware(http.HandlerFunc(h.postCreateRoom)))
    mux.Handle("POST /_matrix/client/v3/rooms/{roomId}/join", authMiddleware(http.HandlerFunc(h.postJoinRoom)))
    mux.Handle("POST /_matrix/client/v3/rooms/{roomId}/leave", authMiddleware(http.HandlerFunc(h.postLeaveRoom)))
}

// getPublicRooms lista as salas públicas do servidor.
// GET /_matrix/client/v3/publicRooms
// Ref: https://spec.matrix.org/v1.18/client-server-api/#get_matrixclientv3publicrooms
func (h *Handler) getPublicRooms(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), util.RequestTimeout)
	defer cancel()

	// Query params: limit e since (paginação conforme o spec)
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // default do spec
	if limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed < 0 {
			util.WriteError(w, http.StatusBadRequest, types.NewErrorResponse(types.M_BAD_JSON, "Invalid limit parameter"))
			return
		}
		limit = parsed
	}
	since := r.URL.Query().Get("since")

	canais, nextBatch, err := h.canalStore.ListPublic(ctx, limit, since)
	if err != nil {
		log.Printf("[ERROR] GET /publicRooms: %v", err)
		util.WriteError(w, http.StatusInternalServerError, types.NewErrorResponse(types.M_UNKNOWN, "Failed to list rooms"))
		return
	}

	chunks := make([]PublicRoomChunk, 0, len(canais))
	for _, canal := range canais {
		jr := canal.JoinRules

		var nome, descricao, foto *string
    		if canal.Nome != "" {
       		 nome = &canal.Nome
   		}
    	if canal.Descricao != "" {
        	descricao = &canal.Descricao
    	}
    	if canal.Foto != "" {
     	   foto = &canal.Foto
    	}

		chunks = append(chunks, PublicRoomChunk{
			RoomID:           canal.ID,
			Name:             nome,
			Topic:            descricao,
			AvatarURL:        foto,
			CanonicalAlias:   canal.CanonAlias,
			NumJoinedMembers: canal.MemberCount,
			WorldReadable:    false,
			GuestCanJoin:     canal.GuestAccess == "can_join",
			JoinRule:         &jr,
		})
	}

	total := len(chunks)
	resp := PublicRoomsResponse{
		Chunk:                  chunks,
		TotalRoomCountEstimate: &total,
	}
	if nextBatch != "" {
		resp.NextBatch = &nextBatch
	}

	util.WriteJSON(w, http.StatusOK, resp)
}

// postCreateRoom cria uma nova sala para o usuário autenticado.
// POST /_matrix/client/v3/createRoom
// Ref: https://spec.matrix.org/v1.18/client-server-api/#post_matrixclientv3createroom
func (h *Handler) postCreateRoom(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), util.RequestTimeout)
	defer cancel()

	// Mesmo padrão de postLogout: lê user_id do contexto injetado pelo middleware de auth
	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		util.WriteError(w, http.StatusUnauthorized, types.NewErrorResponse(types.M_MISSING_TOKEN, "Missing or invalid access token"))
		return
	}

	var req CreateRoomRequest
	if err := util.ParseBody(r, &req); err != nil {
		if err == types.ErrBodyRequired {
			util.WriteError(w, http.StatusBadRequest, types.NewErrorResponse(types.M_NOT_JSON, "No request body"))
		} else {
			util.WriteError(w, http.StatusBadRequest, types.NewErrorResponse(types.M_BAD_JSON, "Invalid request body"))
		}
		return
	}

	// Determina visibilidade e join_rules conforme o spec:
	// preset sobrescreve visibility quando presente
	isPublic := req.Visibility == "public"
	joinRules := "invite"
	if isPublic {
		joinRules = "public"
	}
	if req.Preset != nil {
		switch *req.Preset {
		case "public_chat":
			isPublic = true
			joinRules = "public"
		case "private_chat", "trusted_private_chat":
			isPublic = false
			joinRules = "invite"
		}
	}

	version := "11" // versão padrão atual do Matrix
	if req.RoomVersion != nil && *req.RoomVersion != "" {
		version = *req.RoomVersion
	}

	localPart, err := generateRoomLocalPart()
	if err != nil {
		log.Printf("[ERROR] POST /createRoom: failed to generate room id: %v", err)
		util.WriteError(w, http.StatusInternalServerError, types.NewErrorResponse(types.M_UNKNOWN, "Failed to generate room ID"))
		return
	}

	nome := ""
	if req.Name != nil {
    	nome = *req.Name
	}
	descricao := ""
	if req.Topic != nil {
    	descricao = *req.Topic
	}

	canalCreate := &model.CanalCreate{
		LocalPart:   localPart,
		ServerName:  h.serverName,
		Nome:        nome,
		Descricao:   descricao,
		IsPublic:    isPublic,
		JoinRules:   joinRules,
		GuestAccess: "forbidden",
		HistoryVisibility: "shared",
		Versao:     version,
		CriadorID:   userID,
	}
	canal := canalCreate.ToCanal()

	if err := h.canalStore.Create(ctx, &canal); err != nil {
		log.Printf("[ERROR] POST /createRoom: %v", err)
		util.WriteError(w, http.StatusInternalServerError, types.NewErrorResponse(types.M_UNKNOWN, "Failed to create room"))
		return
	}

	// O criador entra automaticamente como membro (spec)
	membership := &model.UsuarioCanal{
		CanalID:     canal.ID,
		UsuarioID:     userID,
		Membresia:   "join",
		JoinedAt:   time.Now(),
	}
	if err := h.usuarioCanalStore.AddOrUpdateMembership(ctx, membership); err != nil {
		log.Printf("[ERROR] POST /createRoom: failed to add creator membership: %v", err)
		// não falha a resposta, a sala foi criada; log é suficiente por ora
	}

	util.WriteJSON(w, http.StatusOK, CreateRoomResponse{RoomID: canal.ID})
}

// postJoinRoom adiciona o usuário autenticado à sala especificada.
// POST /_matrix/client/v3/rooms/{roomId}/join
// Ref: https://spec.matrix.org/v1.18/client-server-api/#post_matrixclientv3roomsroomidjoin
func (h *Handler) postJoinRoom(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), util.RequestTimeout)
	defer cancel()

	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		util.WriteError(w, http.StatusUnauthorized, types.NewErrorResponse(types.M_MISSING_TOKEN, "Missing or invalid access token"))
		return
	}

	// r.PathValue extrai parâmetros de rota do padrão {roomId}
	roomID := r.PathValue("roomId")
	if roomID == "" {
		util.WriteError(w, http.StatusBadRequest, types.NewErrorResponse(types.M_BAD_JSON, "Missing roomId"))
		return
	}

	// body é opcional pelo spec; ignoramos erro de parse
	var req JoinRoomRequest
	_ = util.ParseBody(r, &req)

	room, err := h.canalStore.GetByID(ctx, roomID)
	if err != nil {
		// O spec manda M_FORBIDDEN (não M_NOT_FOUND) para não vazar existência da sala
		util.WriteError(w, http.StatusForbidden, types.NewErrorResponse(types.M_FORBIDDEN, "Room not found or not accessible"))
		return
	}

	// Verifica permissão de entrada conforme join_rules
	if room.JoinRules == "invite" {
		existing, err := h.usuarioCanalStore.GetByComposedID(ctx, userID, roomID)
		if err != nil || existing.Membresia != "invite" {
			util.WriteError(w, http.StatusForbidden, types.NewErrorResponse(types.M_FORBIDDEN, "You are not invited to this room"))
			return
		}
	}

	membership := &model.UsuarioCanal{
		CanalID:     roomID,
		UsuarioID:     userID,
		Membresia: "join",
		JoinedAt:   time.Now(),
	}
	if err := h.usuarioCanalStore.AddOrUpdateMembership(ctx, membership); err != nil {
		log.Printf("[ERROR] POST /rooms/%s/join: %v", roomID, err)
		util.WriteError(w, http.StatusInternalServerError, types.NewErrorResponse(types.M_UNKNOWN, "Failed to join room"))
		return
	}

	if err := h.canalStore.UpdateMemberCount(ctx, roomID, +1); err != nil {
		log.Printf("[ERROR] POST /rooms/%s/join: failed to update member count: %v", roomID, err)
	}

	util.WriteJSON(w, http.StatusOK, JoinRoomResponse{RoomID: roomID})
}

// postLeaveRoom remove o usuário autenticado da sala especificada.
// POST /_matrix/client/v3/rooms/{roomId}/leave
// Ref: https://spec.matrix.org/v1.18/client-server-api/#post_matrixclientv3roomsroomidleave
func (h *Handler) postLeaveRoom(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), util.RequestTimeout)
	defer cancel()

	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		util.WriteError(w, http.StatusUnauthorized, types.NewErrorResponse(types.M_MISSING_TOKEN, "Missing or invalid access token"))
		return
	}

	roomID := r.PathValue("roomId")
	if roomID == "" {
		util.WriteError(w, http.StatusBadRequest, types.NewErrorResponse(types.M_BAD_JSON, "Missing roomId"))
		return
	}

	// body é opcional pelo spec
	var req LeaveRoomRequest
	_ = util.ParseBody(r, &req)

	existing, err := h.usuarioCanalStore.GetByComposedID(ctx, userID, roomID)
	if err != nil || existing.Membresia == "leave" {
		util.WriteError(w, http.StatusForbidden, types.NewErrorResponse(types.M_FORBIDDEN, "You are not a member of this room"))
		return
	}

	membership := &model.UsuarioCanal{
		CanalID:     roomID,
		UsuarioID:    userID,
		Membresia: "leave",
		JoinedAt:   existing.JoinedAt,
	}
	if err := h.usuarioCanalStore.AddOrUpdateMembership(ctx, membership); err != nil {
		log.Printf("[ERROR] POST /rooms/%s/leave: %v", roomID, err)
		util.WriteError(w, http.StatusInternalServerError, types.NewErrorResponse(types.M_UNKNOWN, "Failed to leave room"))
		return
	}

	if err := h.canalStore.UpdateMemberCount(ctx, roomID, -1); err != nil {
		log.Printf("[ERROR] POST /rooms/%s/leave: failed to update member count: %v", roomID, err)
	}

	// Spec exige {} com 200 OK — mesmo padrão do postLogout
	util.WriteJSON(w, http.StatusOK, map[string]any{})
}

// generateRoomLocalPart gera a parte local do room_id (!localPart:server).
// Mesmo padrão de GenerateRefreshToken em jwt.go: crypto/rand + base64.
func generateRoomLocalPart() (string, error) {
	bytes := make([]byte, 18)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// RawURLEncoding: sem padding '=', seguro para IDs Matrix
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}