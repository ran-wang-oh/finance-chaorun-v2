CREATE TABLE IF NOT EXISTS journal_lines (
  id                TEXT PRIMARY KEY,
  entity_id         TEXT NOT NULL,
  journal_entry_id  TEXT NOT NULL REFERENCES journal_entries(id),
  account_id        TEXT NOT NULL,
  account_code      TEXT NOT NULL,
  account_name      TEXT NOT NULL,
  direction         TEXT NOT NULL,
  debit_amount      NUMERIC(18,2) NOT NULL DEFAULT 0,
  credit_amount     NUMERIC(18,2) NOT NULL DEFAULT 0,
  currency          TEXT NOT NULL DEFAULT 'CNY',
  line_no           INTEGER NOT NULL,
  auxiliary         JSONB NOT NULL DEFAULT '{}',
  UNIQUE (entity_id, journal_entry_id, line_no)
);
