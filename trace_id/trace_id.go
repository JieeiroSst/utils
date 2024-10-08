package trace_id

import (
	"context"

	"go.opentelemetry.io/otel"
)

func TracerID(name string) context.Context {
	tracer := otel.Tracer(name)
	ctx, span := tracer.Start(context.Background(), "root-span")
	defer span.End()
	return ctx
}
