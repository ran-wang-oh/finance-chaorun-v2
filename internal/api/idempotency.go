package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"finance.chao.run/v2/internal/store"
)

func HashInput(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

type IdempotencyStore interface {
	Get(ctx context.Context, entityID, capabilityID, idempotencyKey string) (*store.IdempotencyRecord, error)
	Save(ctx context.Context, r *store.IdempotencyRecord) error
}
