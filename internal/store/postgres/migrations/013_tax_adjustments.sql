CREATE TABLE IF NOT EXISTS tax_adjustments (
  id          TEXT PRIMARY KEY,
  entity_id   TEXT NOT NULL,
  book_id     TEXT NOT NULL DEFAULT '',
  tax_year    TEXT NOT NULL,
  category    TEXT NOT NULL,
  book_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
  tax_base    NUMERIC(18,2) NOT NULL DEFAULT 0,
  adjustment  NUMERIC(18,2) NOT NULL DEFAULT 0,
  formula     TEXT NOT NULL DEFAULT '',
  detail      TEXT NOT NULL DEFAULT '',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (entity_id, tax_year, category, id)
);

CREATE INDEX IF NOT EXISTS idx_tax_adj_entity ON tax_adjustments(entity_id, tax_year);
