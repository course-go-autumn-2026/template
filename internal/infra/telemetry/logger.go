package telemetry

import (
	"context"
	"io"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

type traceHandler struct{ handler slog.Handler }

func NewLogger(output io.Writer, level slog.Level) *slog.Logger {
	return slog.New(&traceHandler{handler: slog.NewJSONHandler(output, &slog.HandlerOptions{Level: level})}).With("service", "trip-service")
}

func (h *traceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *traceHandler) Handle(ctx context.Context, record slog.Record) error {
	span := trace.SpanContextFromContext(ctx)
	traceID, spanID := "", ""

	if span.IsValid() {
		traceID = span.TraceID().String()
		spanID = span.SpanID().String()
	}

	record.AddAttrs(slog.String("trace_id", traceID), slog.String("span_id", spanID))

	return h.handler.Handle(ctx, record)
}

func (h *traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *traceHandler) WithGroup(name string) slog.Handler {
	return &traceHandler{handler: h.handler.WithGroup(name)}
}
