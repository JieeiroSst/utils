package trace_id

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func TracerID(name string) context.Context {
	tracer := otel.Tracer(name)
	ctx, span := tracer.Start(context.Background(), "root-span")
	defer span.End()
	return ctx
}

func ToString(ctx context.Context) string {
	return trace.SpanFromContext(ctx).SpanContext().TraceID().String()
}
