package trace_id

import (
	"context"

	"github.com/google/uuid"
)

const (
	Key = "TRANSACTION_ID_KEY"
)

func generateTransactionID() string {
	return uuid.New().String()
}

func TracerID(ctx context.Context) context.Context {
	tid := generateTransactionID()
	return context.WithValue(ctx, Key, tid)
}

func ToString(ctx context.Context) string {
	tid := ctx.Value(Key)
	return tid.(string)
}
