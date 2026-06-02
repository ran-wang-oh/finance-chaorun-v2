CREATE TABLE IF NOT EXISTS logistics_records (
  id            TEXT PRIMARY KEY,
  entity_id     TEXT NOT NULL,
  invoice_id    TEXT NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
  waybill_no    TEXT NOT NULL,
  carrier       TEXT NOT NULL DEFAULT '',
  status        TEXT NOT NULL DEFAULT 'shipped',
  ship_date     DATE,
  delivery_date DATE,
  items         TEXT NOT NULL DEFAULT '',
  notes         TEXT NOT NULL DEFAULT '',
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_logistics_invoice ON logistics_records(invoice_id);
CREATE INDEX IF NOT EXISTS idx_logistics_entity ON logistics_records(entity_id);
CREATE INDEX IF NOT EXISTS idx_logistics_status ON logistics_records(status);
