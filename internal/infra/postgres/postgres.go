package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	URL            string
	MaxConns       int32
	MinConns       int32
	MaxConnLife    time.Duration
	ConnectTimeout time.Duration
	DBDurationHist metric.Float64Histogram
}

type queryStartKey struct{}

type dbTracer struct {
	otelTracer *otelpgx.Tracer
	hist       metric.Float64Histogram
}

type queryTraceInfo struct {
	start time.Time
	op    string
	coll  string
}

func (t *dbTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx = t.otelTracer.TraceQueryStart(ctx, conn, data)
	op, coll := parseSQLMetadata(data.SQL)

	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		if op != "" {
			span.SetAttributes(attribute.String("db.operation.name", op))
		}
		if coll != "" {
			span.SetAttributes(attribute.String("db.collection.name", coll))
		}
	}

	return context.WithValue(ctx, queryStartKey{}, queryTraceInfo{
		start: time.Now(),
		op:    op,
		coll:  coll,
	})
}

func (t *dbTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	t.otelTracer.TraceQueryEnd(ctx, conn, data)

	info, ok := ctx.Value(queryStartKey{}).(queryTraceInfo)
	if !ok {
		return
	}

	var errType string
	if data.Err != nil {
		errType = classifyDBError(data.Err)
		span := trace.SpanFromContext(ctx)
		if span.IsRecording() {
			span.SetStatus(codes.Error, data.Err.Error())
			span.SetAttributes(attribute.String("error.type", errType))
		}
	}

	if t.hist != nil && info.op != "" {
		attrs := []attribute.KeyValue{
			attribute.String("db.operation.name", info.op),
			attribute.String("db.collection.name", info.coll),
			attribute.String("error.type", errType),
		}
		t.hist.Record(ctx, time.Since(info.start).Seconds(), metric.WithAttributes(attrs...))
	}
}

func classifyDBError(err error) string {
	if errors.Is(err, pgx.ErrNoRows) {
		return "no_rows"
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return "db_error"
}

func parseSQLMetadata(sql string) (string, string) {
	fields := strings.Fields(strings.ToUpper(strings.TrimSpace(sql)))
	if len(fields) == 0 {
		return "", ""
	}
	op := fields[0]
	var coll string
	switch op {
	case "INSERT":
		for i := 1; i < len(fields)-1; i++ {
			if fields[i] == "INTO" {
				coll = strings.ToLower(strings.Trim(fields[i+1], `"`))
				break
			}
		}
	case "UPDATE":
		if len(fields) > 1 {
			coll = strings.ToLower(strings.Trim(fields[1], `"`))
		}
	case "SELECT", "DELETE":
		for i := 1; i < len(fields)-1; i++ {
			if fields[i] == "FROM" {
				coll = strings.ToLower(strings.Trim(fields[i+1], `"`))
				break
			}
		}
	}
	return op, coll
}

func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse db url: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLife
	poolCfg.ConnConfig.Tracer = &dbTracer{
		otelTracer: otelpgx.NewTracer(),
		hist:       cfg.DBDurationHist,
	}

	connectCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(connectCtx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return pool, nil
}
