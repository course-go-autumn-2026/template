package transaction

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PGXBeginner struct {
	pool *pgxpool.Pool
}

func NewPGXBeginner(pool *pgxpool.Pool) *PGXBeginner {
	return &PGXBeginner{pool: pool}
}

func (b *PGXBeginner) Begin(ctx context.Context) (Tx, error) {
	return b.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
}