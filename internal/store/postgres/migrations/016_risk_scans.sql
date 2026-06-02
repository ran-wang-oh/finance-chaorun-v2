CREATE TABLE IF NOT EXISTS risk_scans (
  id              TEXT PRIMARY KEY,
  entity_id       TEXT NOT NULL,
  book_id         TEXT NOT NULL DEFAULT '',
  period          TEXT NOT NULL,
  scan_type       TEXT NOT NULL DEFAULT 'full',
  rules_triggered JSONB NOT NULL DEFAULT '[]',
  findings        JSONB NOT NULL DEFAULT '[]',
  total_score     NUMERIC(10,2) NOT NULL DEFAULT 0,
  risk_level      TEXT NOT NULL DEFAULT 'low',
  engine_version  TEXT NOT NULL DEFAULT '',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_risk_scans_entity ON risk_scans(entity_id, book_id, period);
