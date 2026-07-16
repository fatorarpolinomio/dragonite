package postgres

import (
	"context"
	"fmt"

	"github.com/caio-bernardo/dragonite/internal/domain"
	"github.com/jackc/pgx/v5"
)

// CreateBackupVersion insere uma nova versão de backup de chaves para o usuário
func (s *PostgresStorage) CreateBackupVersion(ctx context.Context, backup *domain.VersaoBackup) error {
	row := s.db.QueryRow(ctx, `
		INSERT INTO VersaoBackup (id_usuario, algorithm, auth_data, count, etag)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id_versao, created_at
	`, backup.IDUsuario, backup.Algorithm, backup.AuthData, backup.Count, backup.ETag)

	if err := row.Scan(&backup.IDVersao, &backup.CreatedAt); err != nil {
		return fmt.Errorf("failure to create backup version for user '%s': %w", backup.IDUsuario, err)
	}

	return nil
}

// GetLatestBackupVersion recupera a versão de backup mais recente (maior id_versao) do usuário
// Retorna (nil, nil) se o usuário nunca criou um backup
func (s *PostgresStorage) GetLatestBackupVersion(ctx context.Context, userID string) (*domain.VersaoBackup, error) {
	row := s.db.QueryRow(ctx, `
		SELECT id_versao, id_usuario, algorithm, auth_data, count, etag, created_at
		FROM VersaoBackup
		WHERE id_usuario = $1
		ORDER BY id_versao DESC
		LIMIT 1
	`, userID)

	var backup domain.VersaoBackup
	err := row.Scan(
		&backup.IDVersao,
		&backup.IDUsuario,
		&backup.Algorithm,
		&backup.AuthData,
		&backup.Count,
		&backup.ETag,
		&backup.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch latest backup version for user '%s': %w", userID, err)
	}

	return &backup, nil
}