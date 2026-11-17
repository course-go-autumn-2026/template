package transaction

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Executor interface {
	Exec(ctx context.Context, sqlQuery string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sqlQuery string, args ...any) pgx.Row
	Query(ctx context.Context, sqlQuery string, args ...any) (pgx.Rows, error)
}

func ExecutorFromContext(ctx context.Context, pool Executor) Executor {
	value, ok := ctx.Value(txKey{}).(pgx.Tx)
	if ok {
		return value
	}

	return pool
}
