package postgres

import (
	"context"

	"finance.chao.run/v2/internal/store"

	"github.com/jackc/pgx/v5"
)

type IdempotencyStore struct {
	db *DB
}

func NewIdempotencyStore(db *DB) *IdempotencyStore {
	return &IdempotencyStore{db: db}
}

func (s *IdempotencyStore) Get(ctx context.Context, entityID, capabilityID, idempotencyKey string) (*store.IdempotencyRecord, error) {
	row := s.db.Pool.QueryRow(ctx, `
		SELECT entity_id, capability_id, idempotency_key, input_hash, result, status,
		       to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM idempotency_records WHERE entity_id = $1 AND capability_id = $2 AND idempotency_key = $3
	`, entityID, capabilityID, idempotencyKey)

	var r store.IdempotencyRecord
	err := row.Scan(&r.EntityID, &r.CapabilityID, &r.IdempotencyKey, &r.InputHash, &r.Result, &r.Status, &r.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

func (s *IdempotencyStore) Save(ctx context.Context, r *store.IdempotencyRecord) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO idempotency_records (entity_id, capability_id, idempotency_key, input_hash, result, status)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (entity_id, capability_id, idempotency_key) DO NOTHING
	`, r.EntityID, r.CapabilityID, r.IdempotencyKey, r.InputHash, r.Result, r.Status)
	return err
}
