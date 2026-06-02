CREATE TABLE IF NOT EXISTS chart_of_accounts (
  id             TEXT PRIMARY KEY,
  entity_id      TEXT NOT NULL,
  book_id        TEXT NOT NULL,
  code           TEXT NOT NULL,
  name           TEXT NOT NULL,
  category       TEXT NOT NULL,
  balance_type   TEXT NOT NULL,
  parent_id      TEXT,
  is_system      BOOLEAN NOT NULL DEFAULT false,
  tax_relevant   BOOLEAN NOT NULL DEFAULT false,
  keywords       TEXT[] NOT NULL DEFAULT '{}',
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (entity_id, book_id, code)
);
