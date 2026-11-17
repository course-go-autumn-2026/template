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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

type Tracer struct {
	base     *otelpgx.Tracer
	duration metric.Float64Histogram
}

type (
	queryKey struct{}

	query struct {
		start      time.Time
		operation  string
		collection string
	}
)

func NewTracer(tp trace.TracerProvider, mp metric.MeterProvider) (*Tracer, error) {
	duration, err := mp.Meter("tripgo.postgres").Float64Histogram("db.client.operation.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, fmt.Errorf("database histogram: %w", err)
	}

	return &Tracer{base: otelpgx.NewTracer(otelpgx.WithTracerProvider(tp), otelpgx.WithMeterProvider(noop.NewMeterProvider()), otelpgx.WithDisableSQLStatementInAttributes(), otelpgx.WithDisableConnectionDetailsInAttributes()), duration: duration}, nil
}

func (t *Tracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx = t.base.TraceQueryStart(ctx, conn, data)
	operation := otelpgx.SQLOperationName(ctx, data.SQL)
	collection := ""

	for _, table := range []string{"trip_status_history", "trips"} {
		if strings.Contains(strings.ToLower(data.SQL), table) {
			collection = table
			break
		}
	}

	trace.SpanFromContext(ctx).SetAttributes(attribute.String("db.operation.name", operation), attribute.String("db.collection.name", collection))

	return context.WithValue(ctx, queryKey{}, query{start: time.Now(), operation: operation, collection: collection})
}

func (t *Tracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	q, ok := ctx.Value(queryKey{}).(query)
	if !ok {
		return
	}

	errorType := ""

	if data.Err != nil {
		errorType = fmt.Sprintf("%T", data.Err)

		var pgErr *pgconn.PgError

		if errors.As(data.Err, &pgErr) {
			errorType = pgErr.Code
		}

		span := trace.SpanFromContext(ctx)
		span.SetAttributes(attribute.String("error.type", errorType))
		span.SetStatus(codes.Error, "database operation failed")
	}

	t.duration.Record(ctx, time.Since(q.start).Seconds(), metric.WithAttributes(attribute.String("db.operation.name", q.operation), attribute.String("db.collection.name", q.collection), attribute.String("error.type", errorType)))
	t.base.TraceQueryEnd(ctx, conn, data)
}
