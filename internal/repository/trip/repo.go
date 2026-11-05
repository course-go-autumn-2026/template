package trip

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryImpl struct {
	pool *pgxpool.Pool
	sb   sq.StatementBuilderType
}

func New(pool *pgxpool.Pool) *RepositoryImpl {
	return &RepositoryImpl{
		pool: pool,
		sb:   sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}
