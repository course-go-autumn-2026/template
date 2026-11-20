package telemetry

import (
	"context"
	"io"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

type TraceHandler struct {
	slog.Handler
}

func NewTraceHandler(h slog.Handler) *TraceHandler {
	return &TraceHandler{Handler: h}
}

func (h *TraceHandler) Handle(ctx context.Context, r slog.Record) error {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		r.Add(
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, r)
}

func (h *TraceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TraceHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *TraceHandler) WithGroup(name string) slog.Handler {
	return &TraceHandler{Handler: h.Handler.WithGroup(name)}
}

func NewLogger(level slog.Level, serviceName string) *slog.Logger {
	return NewLoggerWithWriter(os.Stdout, level, serviceName)
}

func NewLoggerWithWriter(w io.Writer, level slog.Level, serviceName string) *slog.Logger {
	jsonHandler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
	})
	traceHandler := NewTraceHandler(jsonHandler)
	return slog.New(traceHandler).With("service", serviceName)
}
