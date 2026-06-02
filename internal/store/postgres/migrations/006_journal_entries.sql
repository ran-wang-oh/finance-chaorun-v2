CREATE TABLE IF NOT EXISTS journal_entries (
  id              TEXT PRIMARY KEY,
  entity_id       TEXT NOT NULL,
  book_id         TEXT NOT NULL,
  period          TEXT NOT NULL,
  voucher_no      TEXT,
  voucher_word    TEXT NOT NULL DEFAULT '记',
  entry_date      DATE NOT NULL,
  summary         TEXT NOT NULL,
  source_type     TEXT,
  source_id       TEXT,
  status          TEXT NOT NULL,
  created_by      TEXT,
  posted_by       TEXT,
  posted_at       TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (entity_id, book_id, period, voucher_no)
);
