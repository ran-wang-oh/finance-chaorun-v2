CREATE TABLE IF NOT EXISTS finance_audit_log (
  id                  TEXT PRIMARY KEY,
  entity_id           TEXT NOT NULL,
  book_id             TEXT,
  capability_id       TEXT NOT NULL,
  v2_capability_id    TEXT,
  actor_type          TEXT NOT NULL,
  actor_id            TEXT NOT NULL,
  trace_id            TEXT NOT NULL,
  workflow_run_id     TEXT,
  approval_grant_id   TEXT,
  idempotency_key     TEXT,
  object_type         TEXT,
  object_id           TEXT,
  action              TEXT NOT NULL,
  outcome             TEXT NOT NULL,
  payload             JSONB NOT NULL DEFAULT '{}',
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_finance_audit_entity ON finance_audit_log(entity_id);
CREATE INDEX IF NOT EXISTS idx_finance_audit_object ON finance_audit_log(object_type, object_id);
CREATE INDEX IF NOT EXISTS idx_finance_audit_trace ON finance_audit_log(trace_id);
