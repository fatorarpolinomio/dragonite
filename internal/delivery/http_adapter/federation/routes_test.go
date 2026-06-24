package federation

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
    "strings"
	"testing"
	"time"

	"github.com/caio-bernardo/dragonite/internal/domain"
	"github.com/caio-bernardo/dragonite/internal/usecase"
	"github.com/caio-bernardo/dragonite/internal/util"
)

type fakeSystemStorage struct{}

func (s *fakeSystemStorage) PingDB() map[string]string {
	return map[string]string{"status": "up"}
}

func TestFederationGetVersion(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	sys := usecase.NewSystemService("example.com", "1.0.0", pub, priv, "ed25519:1", &fakeSystemStorage{})
	h := NewHandler(sys, nil, nil, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/_matrix/federation/v1/version", nil)

	h.getVersion(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp VersionResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Server.Name != "example.com" || resp.Server.Version != "1.0.0" {
		t.Fatalf("unexpected server info: %+v", resp.Server)
	}
}

func TestFederationGetServerKeySignature(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	sys := usecase.NewSystemService("example.com", "1.0.0", pub, priv, "ed25519:1", &fakeSystemStorage{})
	h := NewHandler(sys, nil, nil, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/_matrix/key/v2/server", nil)

	before := time.Now()
	h.getServerKey(rec, req)
	after := time.Now()

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp ServerKeyResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.ServerName != "example.com" {
		t.Fatalf("expected server_name example.com, got %s", resp.ServerName)
	}
	if resp.ValidUntilTS <= before.UnixMilli() || resp.ValidUntilTS <= after.UnixMilli() {
		t.Fatalf("expected valid_until_ts in the future")
	}

	sig := resp.Signatures["example.com"]["ed25519:1"]
	sigBytes, err := base64.RawStdEncoding.DecodeString(sig)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}

	resp.Signatures = nil
	canonical, err := util.CanonicalJSON(resp)
	if err != nil {
		t.Fatalf("canonical json: %v", err)
	}

	if !ed25519.Verify(pub, canonical, sigBytes) {
		t.Fatalf("expected signature to verify")
	}
}

// fakeUsuarioStorage implementa usecase.UsuarioStorage para testes de federation
// Apenas GetProfileByID é relevante aqui; os demais são stubs
type fakeUsuarioStorage struct {
    profiles map[string]*domain.Profile
}

func newFakeUsuarioStorage(profiles ...*domain.Profile) *fakeUsuarioStorage {
    m := &fakeUsuarioStorage{profiles: make(map[string]*domain.Profile)}
    for _, p := range profiles {
        m.profiles[p.IDUsuario] = p
    }
    return m
}

func (f *fakeUsuarioStorage) GetProfileByID(ctx context.Context, userID string) (*domain.Profile, error) {
    p, ok := f.profiles[userID]
    if !ok {
        return nil, nil // storage retorna nil, nil quando não encontra
    }
    return p, nil
}

func (f *fakeUsuarioStorage) CreateUsuarioAndProfile(ctx context.Context, u domain.Usuario) (*domain.Usuario, error) { return nil, nil }
func (f *fakeUsuarioStorage) GetUsuarioByID(ctx context.Context, userID string) (*domain.Usuario, error)           { return nil, nil }
func (f *fakeUsuarioStorage) UpdateProfile(ctx context.Context, p domain.Profile) error                            { return nil }
func (f *fakeUsuarioStorage) SearchProfiles(ctx context.Context, f2 usecase.SearchFilter) ([]domain.Profile, error) { return nil, nil }
func (f *fakeUsuarioStorage) AddDirectMessage(ctx context.Context, senderID, receiverID, roomID string) error      { return nil }

// helper para construir o handler de federation com profileService injetado
func newTestHandlerWithProfile(t *testing.T, storage *fakeUsuarioStorage) *Handler {
    t.Helper()
    pub, priv, _ := ed25519.GenerateKey(rand.Reader)
    sys := usecase.NewSystemService("dragonite.com", "1.0.0", pub, priv, "ed25519:1", &fakeSystemStorage{})
    profileSvc := usecase.NewProfileService(storage)
    return NewHandler(sys, nil, nil, profileSvc, nil)
}

func TestGetProfile_MissingUserID(t *testing.T) {
    h := newTestHandlerWithProfile(t, newFakeUsuarioStorage())

    rec := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/_matrix/federation/v1/query/profile", nil)
    h.getProfile(rec, req)

    if rec.Code != http.StatusBadRequest {
        t.Fatalf("expected 400, got %d", rec.Code)
    }
}

func TestGetProfile_NonLocalUser(t *testing.T) {
    h := newTestHandlerWithProfile(t, newFakeUsuarioStorage())

    rec := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet,
        "/_matrix/federation/v1/query/profile?user_id=@alice:matrix.org", nil)
    h.getProfile(rec, req)

    if rec.Code != http.StatusNotFound {
        t.Fatalf("expected 404, got %d", rec.Code)
    }
}

func TestGetProfile_InvalidField(t *testing.T) {
    h := newTestHandlerWithProfile(t, newFakeUsuarioStorage())

    rec := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet,
        "/_matrix/federation/v1/query/profile?user_id=@alice:dragonite.com&field=invalid", nil)
    h.getProfile(rec, req)

    if rec.Code != http.StatusBadRequest {
        t.Fatalf("expected 400, got %d", rec.Code)
    }
}

func TestGetProfile_UserNotFound(t *testing.T) {
    h := newTestHandlerWithProfile(t, newFakeUsuarioStorage()) // store vazio

    rec := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet,
        "/_matrix/federation/v1/query/profile?user_id=@alice:dragonite.com", nil)
    h.getProfile(rec, req)

    if rec.Code != http.StatusNotFound {
        t.Fatalf("expected 404, got %d", rec.Code)
    }
}

func TestGetProfile_FullProfile(t *testing.T) {
    displayName := "Alice"
    avatarURL := "mxc://dragonite.com/abc123"
    profile := &domain.Profile{
        IDUsuario:   "@alice:dragonite.com",
        DisplayName: &displayName,
        AvatarURL:   &avatarURL,
    }
    h := newTestHandlerWithProfile(t, newFakeUsuarioStorage(profile))

    rec := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet,
        "/_matrix/federation/v1/query/profile?user_id=@alice:dragonite.com", nil)
    h.getProfile(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rec.Code)
    }

    var body domain.Profile
    if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
        t.Fatalf("decode: %v", err)
    }
    if body.DisplayName == nil || *body.DisplayName != "Alice" {
        t.Errorf("expected displayname 'Alice', got %v", body.DisplayName)
    }
    if body.AvatarURL == nil || *body.AvatarURL != "mxc://dragonite.com/abc123" {
        t.Errorf("expected avatar_url 'mxc://dragonite.com/abc123', got %v", body.AvatarURL)
    }
}

func TestGetProfile_OnlyDisplayName(t *testing.T) {
    displayName := "Alice"
    avatarURL := "mxc://dragonite.com/abc123"
    profile := &domain.Profile{
        IDUsuario:   "@alice:dragonite.com",
        DisplayName: &displayName,
        AvatarURL:   &avatarURL,
    }
    h := newTestHandlerWithProfile(t, newFakeUsuarioStorage(profile))

    rec := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet,
        "/_matrix/federation/v1/query/profile?user_id=@alice:dragonite.com&field=displayname", nil)
    h.getProfile(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rec.Code)
    }

    var body domain.Profile
    if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
        t.Fatalf("decode: %v", err)
    }
    if body.DisplayName == nil || *body.DisplayName != "Alice" {
        t.Errorf("expected displayname 'Alice', got %v", body.DisplayName)
    }
    if body.AvatarURL != nil {
        t.Errorf("expected avatar_url absent, got %v", *body.AvatarURL)
    }
}

func TestGetProfile_OnlyAvatarURL(t *testing.T) {
    displayName := "Alice"
    avatarURL := "mxc://dragonite.com/abc123"
    profile := &domain.Profile{
        IDUsuario:   "@alice:dragonite.com",
        DisplayName: &displayName,
        AvatarURL:   &avatarURL,
    }
    h := newTestHandlerWithProfile(t, newFakeUsuarioStorage(profile))

    rec := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet,
        "/_matrix/federation/v1/query/profile?user_id=@alice:dragonite.com&field=avatar_url", nil)
    h.getProfile(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rec.Code)
    }

    var body domain.Profile
    if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
        t.Fatalf("decode: %v", err)
    }
    if body.AvatarURL == nil || *body.AvatarURL != "mxc://dragonite.com/abc123" {
        t.Errorf("expected avatar_url 'mxc://dragonite.com/abc123', got %v", body.AvatarURL)
    }
    if body.DisplayName != nil {
        t.Errorf("expected displayname absent, got %v", *body.DisplayName)
    }
}

// fakeDirectoryStorage implementa usecase.DirectoryStorage para testes
type fakeDirectoryStorage struct {
    entries []domain.PublicRoomEntry
    total   int
}

func (f *fakeDirectoryStorage) SearchDirectory(_ context.Context, _ string, limit, offset int) ([]domain.PublicRoomEntry, int, error) {
    if offset >= len(f.entries) {
        return []domain.PublicRoomEntry{}, f.total, nil
    }
    result := f.entries[offset:]
    if limit > 0 && len(result) > limit {
        result = result[:limit]
    }
    return result, f.total, nil
}

func newTestHandlerWithDir(t *testing.T, storage *fakeDirectoryStorage) *Handler {
    t.Helper()
    pub, priv, _ := ed25519.GenerateKey(rand.Reader)
    sys := usecase.NewSystemService("dragonite.com", "1.0.0", pub, priv, "ed25519:1", &fakeSystemStorage{})
    dirSvc := usecase.NewDirectoryService(storage, nil, nil)
    return NewHandler(sys, nil, nil, nil, dirSvc)
}

func TestGetPublicRooms_Empty(t *testing.T) {
    h := newTestHandlerWithDir(t, &fakeDirectoryStorage{total: 0})

    rec := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/_matrix/federation/v1/publicRooms", nil)
    h.getPublicRooms(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rec.Code)
    }

    var body domain.PublicRoomsChunck
    if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
        t.Fatalf("decode: %v", err)
    }
    if len(body.Chunk) != 0 {
        t.Errorf("expected empty chunk, got %d items", len(body.Chunk))
    }
    if body.NextBatch != "" {
        t.Errorf("expected no next_batch, got %q", body.NextBatch)
    }
}

func TestGetPublicRooms_WithRooms(t *testing.T) {
    name1 := "General"
    entries := []domain.PublicRoomEntry{
        {RoomID: "!room1:dragonite.com", Name: &name1, NumJoinedMembers: 10, GuestCanJoin: true, WorldReadable: true},
        {RoomID: "!room2:dragonite.com", NumJoinedMembers: 3},
    }
    h := newTestHandlerWithDir(t, &fakeDirectoryStorage{entries: entries, total: 2})

    rec := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/_matrix/federation/v1/publicRooms", nil)
    h.getPublicRooms(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rec.Code)
    }

    var body domain.PublicRoomsChunck
    if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
        t.Fatalf("decode: %v", err)
    }
    if len(body.Chunk) != 2 {
        t.Fatalf("expected 2 rooms, got %d", len(body.Chunk))
    }
    if body.Chunk[0].RoomID != "!room1:dragonite.com" {
        t.Errorf("expected room1 first, got %s", body.Chunk[0].RoomID)
    }
    if !body.Chunk[0].GuestCanJoin {
        t.Error("expected guest_can_join true")
    }
}

func TestGetPublicRooms_Pagination(t *testing.T) {
    entries := []domain.PublicRoomEntry{
        {RoomID: "!r1:dragonite.com", NumJoinedMembers: 3},
        {RoomID: "!r2:dragonite.com", NumJoinedMembers: 2},
        {RoomID: "!r3:dragonite.com", NumJoinedMembers: 1},
    }
    h := newTestHandlerWithDir(t, &fakeDirectoryStorage{entries: entries, total: 3})

    rec := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/_matrix/federation/v1/publicRooms?limit=2", nil)
    h.getPublicRooms(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rec.Code)
    }

    var body domain.PublicRoomsChunck
    if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
        t.Fatalf("decode: %v", err)
    }
    if len(body.Chunk) != 2 {
        t.Fatalf("expected 2 rooms, got %d", len(body.Chunk))
    }
    if body.NextBatch == "" {
        t.Error("expected next_batch to be set")
    }
    if body.PrevBatch != "" {
        t.Errorf("expected no prev_batch on first page, got %q", body.PrevBatch)
    }
}

func TestPostPublicRooms_BadJSON(t *testing.T) {
    h := newTestHandlerWithDir(t, &fakeDirectoryStorage{})

    rec := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodPost, "/_matrix/federation/v1/publicRooms",
        strings.NewReader("{invalid}"))
    h.postPublicRooms(rec, req)

    if rec.Code != http.StatusBadRequest {
        t.Fatalf("expected 400, got %d", rec.Code)
    }
}

func TestPostPublicRooms_WithFilter(t *testing.T) {
    cheeseRoom := "Cheese Lovers"
    entries := []domain.PublicRoomEntry{
        {RoomID: "!cheese:dragonite.com", Name: &cheeseRoom, NumJoinedMembers: 10},
    }
    h := newTestHandlerWithDir(t, &fakeDirectoryStorage{entries: entries, total: 1})

    body, _ := json.Marshal(PublicRoomsRequest{
        Filter: &PublicRoomsFilter{GenericSearchTerm: "cheese"},
        Limit:  10,
    })
    rec := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodPost, "/_matrix/federation/v1/publicRooms",
        strings.NewReader(string(body)))
    h.postPublicRooms(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rec.Code)
    }

    var resp domain.PublicRoomsChunck
    if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
        t.Fatalf("decode: %v", err)
    }
    if len(resp.Chunk) != 1 {
        t.Fatalf("expected 1 room, got %d", len(resp.Chunk))
    }
    if resp.Chunk[0].RoomID != "!cheese:dragonite.com" {
        t.Errorf("expected cheese room, got %s", resp.Chunk[0].RoomID)
    }
}

func TestPostPublicRooms_EmptyBody(t *testing.T) {
    entries := []domain.PublicRoomEntry{
        {RoomID: "!r1:dragonite.com", NumJoinedMembers: 5},
    }
    h := newTestHandlerWithDir(t, &fakeDirectoryStorage{entries: entries, total: 1})

    rec := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodPost, "/_matrix/federation/v1/publicRooms",
        strings.NewReader("{}"))
    h.postPublicRooms(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rec.Code)
    }

    var resp domain.PublicRoomsChunck
    if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
        t.Fatalf("decode: %v", err)
    }
    if len(resp.Chunk) != 1 {
        t.Fatalf("expected 1 room, got %d", len(resp.Chunk))
    }
}