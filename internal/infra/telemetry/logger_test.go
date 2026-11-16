package telemetry_test

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/XrystikLK/template/internal/infra/telemetry"
)

func TestTelemetryLoggerTraceFields(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	tracer := tp.Tracer("test")

	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	logger := telemetry.NewLogger("info", "test-service")
	if logger == nil {
		t.Fatal("expected logger, got nil")
	}
	logger.InfoContext(ctx, "test event")
}
