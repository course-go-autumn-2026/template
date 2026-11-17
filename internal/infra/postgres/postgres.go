package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/freyliee/template/internal/config"
)

func NewPool(ctx context.Context, cfg config.Config, tracer pgx.QueryTracer) (*pgxpool.Pool, error) {
	pc, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid DATABASE_URL")
	}

	pc.MaxConns = cfg.DatabaseMaxConns
	pc.MinConns = cfg.DatabaseMinConns
	pc.MaxConnLifetime = cfg.DatabaseMaxConnLifetime
	pc.ConnConfig.ConnectTimeout = cfg.DatabaseConnectTimeout
	pc.ConnConfig.Tracer = tracer

	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()

		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}
