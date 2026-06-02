CREATE TABLE IF NOT EXISTS tax_profiles (
  id                  TEXT PRIMARY KEY,
  entity_id           TEXT NOT NULL,
  book_id             TEXT NOT NULL,
  taxpayer_type       TEXT NOT NULL DEFAULT 'general',
  vat_taxpayer_type   TEXT NOT NULL DEFAULT 'general',
  cit_rate_type       TEXT NOT NULL DEFAULT 'standard',
  tax_registration_no TEXT NOT NULL DEFAULT '',
  tax_office          TEXT NOT NULL DEFAULT '',
  industry            TEXT NOT NULL DEFAULT '',
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (entity_id, book_id)
);
