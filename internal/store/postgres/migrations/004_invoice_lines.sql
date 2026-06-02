CREATE TABLE IF NOT EXISTS invoice_lines (
  id             TEXT PRIMARY KEY,
  entity_id      TEXT NOT NULL,
  invoice_id     TEXT NOT NULL REFERENCES invoices(id),
  line_no        INTEGER NOT NULL,
  item_name      TEXT NOT NULL,
  item_code      TEXT,
  quantity       NUMERIC(18,4),
  unit_price     NUMERIC(18,4),
  amount         NUMERIC(18,2) NOT NULL,
  tax_rate       NUMERIC(8,4) NOT NULL DEFAULT 0,
  tax_amount     NUMERIC(18,2) NOT NULL DEFAULT 0,
  goods_service_code TEXT,
  metadata       JSONB NOT NULL DEFAULT '{}',
  UNIQUE (entity_id, invoice_id, line_no)
);
