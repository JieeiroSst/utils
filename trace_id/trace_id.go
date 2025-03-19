package trace_id

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey string

const ContextKey ctxKey = "tracer-id"

type TracerID string

func NewTracerID() TracerID {
	return TracerID(uuid.New().String())
}

func (t TracerID) String() string {
	return string(t)
}

func WithTracerID(ctx context.Context, tracerID TracerID) context.Context {
	return context.WithValue(ctx, ContextKey, tracerID)
}

func GetTracerID(ctx context.Context) TracerID {
	id, ok := ctx.Value(ContextKey).(TracerID)
	if !ok {
		return ""
	}
	return id
}

func EnsureTracerID(ctx context.Context) context.Context {
	if id := GetTracerID(ctx); id != "" {
		return ctx
	}
	id := NewTracerID()
	return WithTracerID(ctx, id)
}

func WithContext(ctx context.Context) context.Context {
	ctx = EnsureTracerID(ctx)
	return ctx
}
