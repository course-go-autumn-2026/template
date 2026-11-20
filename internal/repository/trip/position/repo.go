package position

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/George1487/MalyginGeorge/internal/infra/postgres/transaction"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) executor(ctx context.Context) databaseExecutor {
	if tx, ok := transaction.TxFromContext(ctx); ok {
		return tx
	}
	return r.pool
}
