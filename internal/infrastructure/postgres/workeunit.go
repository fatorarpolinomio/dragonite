package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NOTE: used to avoid type conflicts
type contextKey string

type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

const txKey contextKey = "tx"

// Switches between transaction and pool pointer
func getTxOrPool(ctx context.Context, pool *pgxpool.Pool) Querier {
	if tx, ok := ctx.Value(txKey).(pgx.Tx); ok {
		return tx
	}
	return pool
}

// Executes a function under a transaction
func (s *PostgresStorage) Execute(
	ctx context.Context,
	fn func(txCtx context.Context) error,
) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txCtx := context.WithValue(ctx, txKey, tx)

	err = fn(txCtx)
	if err != nil {
		return err
	}
	return tx.Commit(txCtx)
}
