package postgres

import (
	"context"
	"regexp"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type collectionTracer struct {
	*otelpgx.Tracer
	duration metric.Float64Histogram
}

type ctxKey int

const (
	collTableKey ctxKey = iota
	collOpKey
	collStartKey
)

func newCollectionTracer(tr *otelpgx.Tracer) *collectionTracer {
	meter := otel.GetMeterProvider().Meter("trip-service")
	duration, _ := meter.Float64Histogram("db.client.operation.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Duration of database client operations."),
	)
	return &collectionTracer{Tracer: tr, duration: duration}
}

func (t *collectionTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx = t.Tracer.TraceQueryStart(ctx, conn, data)
	table := extractTableName(data.SQL)
	if span := trace.SpanFromContext(ctx); span.IsRecording() && table != "" {
		span.SetAttributes(attribute.String("db.collection.name", table))
	}
	ctx = context.WithValue(ctx, collTableKey, table)
	ctx = context.WithValue(ctx, collOpKey, otelpgx.SQLOperationName(ctx, data.SQL))
	ctx = context.WithValue(ctx, collStartKey, time.Now())
	return ctx
}

func (t *collectionTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	t.Tracer.TraceQueryEnd(ctx, conn, data)
	table, _ := ctx.Value(collTableKey).(string)
	opName, _ := ctx.Value(collOpKey).(string)
	startTime, _ := ctx.Value(collStartKey).(time.Time)
	attrs := []attribute.KeyValue{
		attribute.String("db.operation.name", opName),
	}
	if table != "" {
		attrs = append(attrs, attribute.String("db.collection.name", table))
	}
	if data.Err != nil {
		attrs = append(attrs, attribute.String("error.type", data.Err.Error()))
	}
	if !startTime.IsZero() {
		t.duration.Record(ctx, time.Since(startTime).Seconds(), metric.WithAttributes(attrs...))
	}
}

var tableRegex = regexp.MustCompile(`(?i)(?:INTO|FROM|UPDATE)\s+([a-zA-Z_][a-zA-Z0-9_]*)`)

func extractTableName(sql string) string {
	m := tableRegex.FindStringSubmatch(sql)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}
