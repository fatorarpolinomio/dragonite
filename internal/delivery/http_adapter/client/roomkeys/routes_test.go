package roomkeys

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/caio-bernardo/dragonite/internal/domain"
	"github.com/caio-bernardo/dragonite/internal/domain/types"
	"github.com/caio-bernardo/dragonite/internal/usecase"
)

type fakeBackupStorage struct {
	created   *domain.VersaoBackup
	createErr error
	nextID    int64
	getResult *domain.VersaoBackup
	getErr    error
}

func (f *fakeBackupStorage) CreateBackupVersion(ctx context.Context, backup *domain.VersaoBackup) error {
	if f.createErr != nil {
		return f.createErr
	}
	if f.nextID == 0 {
		f.nextID = 1
	}
	backup.IDVersao = f.nextID
	f.created = backup
	return nil
}

func (f *fakeBackupStorage) GetLatestBackupVersion(ctx context.Context, userID string) (*domain.VersaoBackup, error) {
	return f.getResult, f.getErr
}

func TestGetLatestVersionOK(t *testing.T) {
	existing := &domain.VersaoBackup{
		IDVersao:  1,
		IDUsuario: "@alice:example.com",
		Algorithm: "m.megolm_backup.v1.curve25519-aes-sha2",
		AuthData:  json.RawMessage(`{"public_key":"abcdefg"}`),
		Count:     42,
		ETag:      "anopaquestring",
	}
	store := &fakeBackupStorage{getResult: existing}
	svc := usecase.NewBackupService(store)
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/_matrix/client/v3/room_keys/version", nil)
	req = req.WithContext(context.WithValue(req.Context(), types.UserIDKey, "@alice:example.com"))
	rec := httptest.NewRecorder()

	h.getLatestVersion(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp BackupVersionResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Version != "1" {
		t.Fatalf("expected version '1', got %s", resp.Version)
	}
	if resp.Count != 42 {
		t.Fatalf("expected count 42, got %d", resp.Count)
	}
	if resp.ETag != "anopaquestring" {
		t.Fatalf("expected etag 'anopaquestring', got %s", resp.ETag)
	}
}

func TestGetLatestVersionNotFound(t *testing.T) {
	store := &fakeBackupStorage{getResult: nil}
	svc := usecase.NewBackupService(store)
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/_matrix/client/v3/room_keys/version", nil)
	req = req.WithContext(context.WithValue(req.Context(), types.UserIDKey, "@alice:example.com"))
	rec := httptest.NewRecorder()

	h.getLatestVersion(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}

	var resp struct {
		ErrCode string `json:"errcode"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if resp.ErrCode != "M_NOT_FOUND" {
		t.Fatalf("expected M_NOT_FOUND, got %s", resp.ErrCode)
	}
}

func TestGetLatestVersionMissingAuth(t *testing.T) {
	store := &fakeBackupStorage{}
	svc := usecase.NewBackupService(store)
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/_matrix/client/v3/room_keys/version", nil)
	rec := httptest.NewRecorder()

	h.getLatestVersion(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestCreateVersionOK(t *testing.T) {
	store := &fakeBackupStorage{nextID: 7}
	svc := usecase.NewBackupService(store)
	h := NewHandler(svc)

	body := bytes.NewBufferString(`{"algorithm":"m.megolm_backup.v1.curve25519-aes-sha2","auth_data":{"public_key":"abcdefg"}}`)
	req := httptest.NewRequest(http.MethodPost, "/_matrix/client/v3/room_keys/version", body)
	req = req.WithContext(context.WithValue(req.Context(), types.UserIDKey, "@alice:example.com"))
	rec := httptest.NewRecorder()

	h.createVersion(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp CreateBackupVersionResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Version != "7" {
		t.Fatalf("expected version '7', got %s", resp.Version)
	}
	if store.created == nil {
		t.Fatal("expected backup to be persisted")
	}
	if store.created.IDUsuario != "@alice:example.com" {
		t.Fatalf("expected id_usuario '@alice:example.com', got %s", store.created.IDUsuario)
	}
}

func TestCreateVersionMissingParams(t *testing.T) {
	store := &fakeBackupStorage{}
	svc := usecase.NewBackupService(store)
	h := NewHandler(svc)

	body := bytes.NewBufferString(`{"algorithm":""}`)
	req := httptest.NewRequest(http.MethodPost, "/_matrix/client/v3/room_keys/version", body)
	req = req.WithContext(context.WithValue(req.Context(), types.UserIDKey, "@alice:example.com"))
	rec := httptest.NewRecorder()

	h.createVersion(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var resp struct {
		ErrCode string `json:"errcode"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if resp.ErrCode != "M_MISSING_PARAM" {
		t.Fatalf("expected M_MISSING_PARAM, got %s", resp.ErrCode)
	}
}

func TestCreateVersionInvalidJSON(t *testing.T) {
	store := &fakeBackupStorage{}
	svc := usecase.NewBackupService(store)
	h := NewHandler(svc)

	body := bytes.NewBufferString(`{invalid`)
	req := httptest.NewRequest(http.MethodPost, "/_matrix/client/v3/room_keys/version", body)
	req = req.WithContext(context.WithValue(req.Context(), types.UserIDKey, "@alice:example.com"))
	rec := httptest.NewRecorder()

	h.createVersion(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestCreateVersionStorageError(t *testing.T) {
	store := &fakeBackupStorage{createErr: errors.New("db connection lost")}
	svc := usecase.NewBackupService(store)
	h := NewHandler(svc)

	body := bytes.NewBufferString(`{"algorithm":"m.megolm_backup.v1.curve25519-aes-sha2","auth_data":{"public_key":"abcdefg"}}`)
	req := httptest.NewRequest(http.MethodPost, "/_matrix/client/v3/room_keys/version", body)
	req = req.WithContext(context.WithValue(req.Context(), types.UserIDKey, "@alice:example.com"))
	rec := httptest.NewRecorder()

	h.createVersion(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}