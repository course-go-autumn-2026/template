package telemetry

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

type TraceHandler struct {
	handler slog.Handler
}

func (h *TraceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *TraceHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx != nil {
		span := trace.SpanFromContext(ctx)
		sc := span.SpanContext()
		if sc.IsValid() {
			r.Add(
				slog.String("trace_id", sc.TraceID().String()),
				slog.String("span_id", sc.SpanID().String()),
			)
		}
	}
	return h.handler.Handle(ctx, r)
}

func (h *TraceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TraceHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *TraceHandler) WithGroup(name string) slog.Handler {
	return &TraceHandler{handler: h.handler.WithGroup(name)}
}

func NewLogger(levelStr, serviceName string) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	jsonHandler := slog.NewJSONHandler(os.Stdout, opts)
	traceHandler := &TraceHandler{handler: jsonHandler}

	logger := slog.New(traceHandler)
	if serviceName != "" {
		logger = logger.With(slog.String("service", serviceName))
	}
	return logger
}
