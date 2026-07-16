package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/caio-bernardo/dragonite/internal/domain"
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

func TestBackupServiceCreateBackupVersionSuccess(t *testing.T) {
	store := &fakeBackupStorage{nextID: 3}
	svc := NewBackupService(store)

	authData := json.RawMessage(`{"public_key":"abcdefg"}`)
	result, err := svc.CreateBackupVersion(context.Background(), CreateBackupParams{
		UserID:    "@alice:example.com",
		Algorithm: "m.megolm_backup.v1.curve25519-aes-sha2",
		AuthData:  authData,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.VersionString() != "3" {
		t.Fatalf("expected version '3', got %s", result.VersionString())
	}
	if store.created == nil {
		t.Fatal("expected backup to be persisted")
	}
	if store.created.IDUsuario != "@alice:example.com" {
		t.Fatalf("expected id_usuario '@alice:example.com', got %s", store.created.IDUsuario)
	}
	if store.created.Count != 0 {
		t.Fatalf("expected initial count 0, got %d", store.created.Count)
	}
	if store.created.ETag != "0" {
		t.Fatalf("expected initial etag '0', got %s", store.created.ETag)
	}
}

func TestBackupServiceCreateBackupVersionStorageError(t *testing.T) {
	store := &fakeBackupStorage{createErr: errors.New("db connection lost")}
	svc := NewBackupService(store)

	_, err := svc.CreateBackupVersion(context.Background(), CreateBackupParams{
		UserID:    "@alice:example.com",
		Algorithm: "m.megolm_backup.v1.curve25519-aes-sha2",
		AuthData:  json.RawMessage(`{}`),
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBackupServiceGetLatestBackupVersionSuccess(t *testing.T) {
	existing := &domain.VersaoBackup{
		IDVersao:  5,
		IDUsuario: "@alice:example.com",
		Algorithm: "m.megolm_backup.v1.curve25519-aes-sha2",
		AuthData:  json.RawMessage(`{"public_key":"abcdefg"}`),
		Count:     42,
		ETag:      "anopaquestring",
	}
	store := &fakeBackupStorage{getResult: existing}
	svc := NewBackupService(store)

	result, err := svc.GetLatestBackupVersion(context.Background(), "@alice:example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.VersionString() != "5" {
		t.Fatalf("expected version '5', got %s", result.VersionString())
	}
	if result.Count != 42 {
		t.Fatalf("expected count 42, got %d", result.Count)
	}
}

func TestBackupServiceGetLatestBackupVersionNotFound(t *testing.T) {
	store := &fakeBackupStorage{getResult: nil}
	svc := NewBackupService(store)

	_, err := svc.GetLatestBackupVersion(context.Background(), "@alice:example.com")
	if !errors.Is(err, ErrBackupNotFound) {
		t.Fatalf("expected ErrBackupNotFound, got %v", err)
	}
}

func TestBackupServiceGetLatestBackupVersionStorageError(t *testing.T) {
	store := &fakeBackupStorage{getErr: errors.New("db timeout")}
	svc := NewBackupService(store)

	_, err := svc.GetLatestBackupVersion(context.Background(), "@alice:example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, ErrBackupNotFound) {
		t.Fatal("expected a generic error, not ErrBackupNotFound, when storage fails")
	}
}