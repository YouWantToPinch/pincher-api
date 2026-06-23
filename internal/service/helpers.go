package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

type CTXKey string

// UUIDFromContext attempts to retrieve a validated UUID from context.
func UUIDFromContext(ctx context.Context, key string) uuid.UUID {
	contextKeyValue, ok := ctx.Value(CTXKey(key)).(uuid.UUID)
	if !ok {
		slog.Warn("failed to retrieve key from context", slog.String("key", key))
		return uuid.Nil
	}
	return contextKeyValue
}

/*
// TxnFromContext attempts to retrieve a validated transaction payload from context.
func TxnFromContext(ctx context.Context, key string) *validatedTxnPayload {
	contextKeyValue, ok := ctx.Value(ctxKey(key)).(*validatedTxnPayload)
	if !ok {
		slog.Warn("failed to retrieve key from context", slog.String("key", key))
		return nil
	}
	return contextKeyValue
}
*/
