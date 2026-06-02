CREATE TABLE IF NOT EXISTS consistency_checks (
  id           TEXT PRIMARY KEY,
  entity_id    TEXT NOT NULL,
  book_id      TEXT NOT NULL DEFAULT '',
  period       TEXT NOT NULL,
  check_type   TEXT NOT NULL,
  check_name   TEXT NOT NULL,
  source_value NUMERIC(18,4) NOT NULL DEFAULT 0,
  target_value NUMERIC(18,4) NOT NULL DEFAULT 0,
  difference   NUMERIC(18,4) NOT NULL DEFAULT 0,
  tolerance    NUMERIC(18,4) NOT NULL DEFAULT 0,
  passed       BOOLEAN NOT NULL DEFAULT true,
  detail       TEXT NOT NULL DEFAULT '',
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_consistency_checks_entity ON consistency_checks(entity_id, book_id, period);
