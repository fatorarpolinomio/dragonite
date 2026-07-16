package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/caio-bernardo/dragonite/internal/domain"
)

// ErrBackupNotFound é retornado quando o usuário nunca criou uma versão de backup
var ErrBackupNotFound = errors.New("no backup version found")

// BackupService contém a lógica para as versões de backup de chaves (E2EE room_keys)
type BackupService struct {
	backupStorage BackupStorage
}

func NewBackupService(backupStorage BackupStorage) *BackupService {
	return &BackupService{backupStorage: backupStorage}
}

// CreateBackupParams contém os dados necessários para criar uma nova versão de backup
type CreateBackupParams struct {
	UserID    string
	Algorithm string
	AuthData  json.RawMessage
}

// CreateBackupVersion cria uma nova versão de backup para o usuário
// Uma versão nova NÃO apaga versões anteriores, só passa a ser a "latest"
func (s *BackupService) CreateBackupVersion(ctx context.Context, params CreateBackupParams) (*domain.VersaoBackup, error) {
	backup := &domain.VersaoBackup{
		IDUsuario: params.UserID,
		Algorithm: params.Algorithm,
		AuthData:  params.AuthData,
		Count:     0,
		ETag:      "0",
	}

	if err := s.backupStorage.CreateBackupVersion(ctx, backup); err != nil {
		return nil, fmt.Errorf("failed to create backup version: %w", err)
	}

	return backup, nil
}

// GetLatestBackupVersion retorna a versão de backup mais recente do usuário
func (s *BackupService) GetLatestBackupVersion(ctx context.Context, userID string) (*domain.VersaoBackup, error) {
	backup, err := s.backupStorage.GetLatestBackupVersion(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest backup version: %w", err)
	}
	if backup == nil {
		return nil, ErrBackupNotFound
	}

	return backup, nil
}