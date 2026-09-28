package database

import (
	"context"
	"fmt"
	"time"

	"github.com/imrtlDrifter/template/internal/config"
	"github.com/imrtlDrifter/template/pkg/database/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	Postgres     *pgxpool.Pool
	TxManager    *postgres.TransactionManager
	queryTimeout time.Duration
}

func InitDB(ctx context.Context, cfg *config.Configuration) (*Database, error) {
	pool, err := postgres.ConnectDB(ctx, cfg.PostgreSQL)
	if err != nil {
		return nil, fmt.Errorf("init postgres: %w", err)
	}

	return &Database{
		Postgres:     pool,
		TxManager:    postgres.NewTransactionManager(pool, cfg.PostgreSQL.QueryTimeout),
		queryTimeout: cfg.PostgreSQL.QueryTimeout,
	}, nil
}

func (d *Database) CheckReady(ctx context.Context) error {
	readyCtx, cancel := context.WithTimeout(ctx, d.queryTimeout)
	defer cancel()

	err := d.Postgres.Ping(readyCtx)
	if err != nil {
		return fmt.Errorf("postgres connection not ready: %w", err)
	}

	return nil
}

func (d *Database) Close() {
	if d == nil || d.Postgres == nil {
		return
	}

	d.Postgres.Close()
}
