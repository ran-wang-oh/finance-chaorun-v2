CREATE TABLE IF NOT EXISTS tax_returns (
  id          TEXT PRIMARY KEY,
  entity_id   TEXT NOT NULL,
  book_id     TEXT NOT NULL,
  tax_year    TEXT NOT NULL,
  tax_period  TEXT NOT NULL,
  return_type TEXT NOT NULL,
  payload     JSONB NOT NULL DEFAULT '{}',
  status      TEXT NOT NULL DEFAULT 'draft',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (entity_id, book_id, tax_year, tax_period, return_type)
);
