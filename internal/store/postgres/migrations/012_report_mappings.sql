CREATE TABLE IF NOT EXISTS report_mappings (
  id                  TEXT PRIMARY KEY,
  report_type         TEXT NOT NULL,
  line_code           TEXT NOT NULL,
  line_label          TEXT NOT NULL,
  display_order       INTEGER NOT NULL DEFAULT 0,
  accounting_standard TEXT NOT NULL DEFAULT 'small_business_gaap_cn',
  account_selector    JSONB NOT NULL DEFAULT '{}',
  is_subtotal         BOOLEAN NOT NULL DEFAULT false,
  parent_line_code    TEXT NOT NULL DEFAULT '',
  formula             TEXT NOT NULL DEFAULT '',
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_report_mappings_type ON report_mappings(report_type, accounting_standard);
