CREATE TABLE IF NOT EXISTS bank_transactions (
  id                   TEXT PRIMARY KEY,
  entity_id            TEXT NOT NULL,
  book_id              TEXT NOT NULL DEFAULT '',
  transaction_date     DATE NOT NULL,
  counterparty_name    TEXT NOT NULL,
  counterparty_account TEXT NOT NULL DEFAULT '',
  amount               NUMERIC(12,2) NOT NULL,
  direction            TEXT NOT NULL DEFAULT 'out',
  summary              TEXT NOT NULL DEFAULT '',
  bank_reference       TEXT NOT NULL DEFAULT '',
  matched_invoice_id   TEXT,
  match_confidence     NUMERIC(3,2) NOT NULL DEFAULT 0,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_bank_tx_invoice ON bank_transactions(matched_invoice_id);
CREATE INDEX IF NOT EXISTS idx_bank_tx_entity ON bank_transactions(entity_id);
CREATE INDEX IF NOT EXISTS idx_bank_tx_date ON bank_transactions(transaction_date);
CREATE INDEX IF NOT EXISTS idx_bank_tx_counterparty ON bank_transactions(entity_id, counterparty_name);
