CREATE TABLE IF NOT EXISTS accounting_books (
  id                  TEXT PRIMARY KEY,
  entity_id           TEXT NOT NULL,
  code                TEXT NOT NULL,
  name                TEXT NOT NULL,
  accounting_standard TEXT NOT NULL,
  base_currency       TEXT NOT NULL DEFAULT 'CNY',
  start_period        TEXT NOT NULL,
  is_default          BOOLEAN NOT NULL DEFAULT false,
  status              TEXT NOT NULL DEFAULT 'active',
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (entity_id, code)
);
