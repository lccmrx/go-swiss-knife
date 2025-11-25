package tracer

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	tracer trace.Tracer
)

func New(name string) {
	tracer = otel.Tracer(name)
}

func Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if tracer == nil {
		return ctx, noop.Span{}
	}
	return tracer.Start(ctx, spanName, opts...)
}
