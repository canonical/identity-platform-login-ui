package tracing

import (
	"context"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
	"go.opentelemetry.io/otel/trace/noop"
)

type NoopTracer struct {
	embedded.Tracer
	tracer trace.Tracer

	logger logging.LoggerInterface
}

func NewNoopTracer(cfg *Config) *NoopTracer {
	t := new(NoopTracer)
	t.tracer = new(noop.Tracer)
	return t
}

func (t *NoopTracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, spanName, opts...)
}
