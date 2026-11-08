package trip

import (
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool         *pgxpool.Pool
	sq           squirrel.StatementBuilderType
	queryTimeout time.Duration
}

func New(pool *pgxpool.Pool, queryTimeout time.Duration) *Repository {
	if queryTimeout <= 0 {
		queryTimeout = 3 * time.Second
	}
	return &Repository{
		pool:         pool,
		sq:           squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		queryTimeout: queryTimeout,
	}
}
