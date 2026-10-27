package database

import (
"context"
"fmt"
"time"

"github.com/jackc/pgx/v5/pgxpool"

"github.com/razrabbesezdav-art/template/internal/config"
)

func NewPostgres(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
if err != nil {
return nil, fmt.Errorf("parse database config: %w", err)
}

poolConfig.MaxConns = cfg.DatabaseMaxConns
poolConfig.MinConns = cfg.DatabaseMinConns
poolConfig.MaxConnLifetime = cfg.DatabaseMaxConnLifetime
poolConfig.ConnConfig.ConnectTimeout = cfg.DatabaseConnectTimeout

connectCtx, cancel := context.WithTimeout(ctx, cfg.DatabaseConnectTimeout)
defer cancel()

pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
if err != nil {
return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
}

pingCtx, pingCancel := context.WithTimeout(ctx, cfg.DatabaseConnectTimeout)
defer pingCancel()

if err := pool.Ping(pingCtx); err != nil {
pool.Close()
return nil, fmt.Errorf("ping PostgreSQL: %w", err)
}

// Keep this reference to make the timeout configuration explicit:
// individual queries will use cfg.DatabaseQueryTimeout.
_ = time.Second

return pool, nil
}
