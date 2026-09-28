package postgres

import (
	"context"
	"fmt"

	"github.com/imrtlDrifter/template/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectDB(ctx context.Context, pg config.PostgreSQL) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(pg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.MaxConns = pg.MaxConns
	cfg.MinConns = pg.MinConns
	cfg.MaxConnLifetime = pg.MaxConnLifetime
	cfg.ConnConfig.ConnectTimeout = pg.ConnectTimeout

	connectCtx, cancel := context.WithTimeout(ctx, pg.ConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	return pool, nil
}
