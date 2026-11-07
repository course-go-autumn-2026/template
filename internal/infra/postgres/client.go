package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/XrystikLK/template/internal/config"
)

type dbTracer struct {
	*otelpgx.Tracer
}

func (t *dbTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx = t.Tracer.TraceQueryStart(ctx, conn, data)
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		collection := extractCollection(data.SQL)
		if collection != "" {
			span.SetAttributes(attribute.String("db.collection.name", collection))
		}
	}
	return ctx
}

func (t *dbTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	t.Tracer.TraceQueryEnd(ctx, conn, data)
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() && data.Err != nil && !errors.Is(data.Err, pgx.ErrNoRows) {
		span.SetStatus(codes.Error, data.Err.Error())
		span.SetAttributes(attribute.String("error.type", "internal_error"))
	}
}

func extractCollection(sql string) string {
	lower := strings.ToLower(sql)
	if strings.Contains(lower, "trip_status_history") {
		return "trip_status_history"
	}
	if strings.Contains(lower, "trips") {
		return "trips"
	}
	return ""
}

func NewPool(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга DATABASE_URL: %w", err)
	}

	tracer := otelpgx.NewTracer(otelpgx.WithMetricOperationName(otelpgx.SQLOperationName))
	poolCfg.ConnConfig.Tracer = &dbTracer{Tracer: tracer}

	poolCfg.MaxConns = cfg.DatabaseMaxConns
	poolCfg.MinConns = cfg.DatabaseMinConns
	poolCfg.MaxConnLifetime = cfg.DatabaseMaxConnLifetime
	poolCfg.ConnConfig.ConnectTimeout = cfg.DatabaseConnectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания пула соединений: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.DatabaseConnectTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("база данных недоступна: %w", err)
	}

	return pool, nil
}
