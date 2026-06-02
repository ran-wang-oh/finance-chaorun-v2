CREATE TABLE IF NOT EXISTS accounting_periods (
  id          TEXT PRIMARY KEY,
  entity_id   TEXT NOT NULL,
  book_id     TEXT NOT NULL,
  period      TEXT NOT NULL,
  status      TEXT NOT NULL DEFAULT 'open',
  opened_at   TIMESTAMPTZ,
  closing_at  TIMESTAMPTZ,
  closed_at   TIMESTAMPTZ,
  locked_at   TIMESTAMPTZ,
  closed_by   TEXT,
  metadata    JSONB NOT NULL DEFAULT '{}',
  UNIQUE (entity_id, book_id, period)
);
