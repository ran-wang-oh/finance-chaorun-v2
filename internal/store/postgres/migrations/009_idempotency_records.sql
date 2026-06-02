CREATE TABLE IF NOT EXISTS idempotency_records (
  entity_id        TEXT NOT NULL,
  capability_id    TEXT NOT NULL,
  idempotency_key  TEXT NOT NULL,
  input_hash       TEXT NOT NULL,
  result           JSONB NOT NULL,
  status           TEXT NOT NULL,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (entity_id, capability_id, idempotency_key)
);
