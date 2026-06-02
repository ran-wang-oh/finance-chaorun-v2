package postgres

import (
	"context"

	"finance.chao.run/v2/internal/domain"

	"github.com/jackc/pgx/v5"
)

type InvoiceLineStore struct {
	db *DB
}

func NewInvoiceLineStore(db *DB) *InvoiceLineStore {
	return &InvoiceLineStore{db: db}
}

func (s *InvoiceLineStore) CreateMany(ctx context.Context, lines []domain.InvoiceLine) error {
	return s.CreateManyTx(ctx, nil, lines)
}

func (s *InvoiceLineStore) CreateManyTx(ctx context.Context, tx pgx.Tx, lines []domain.InvoiceLine) error {
	exec := s.db.Pool.Exec
	if tx != nil {
		exec = tx.Exec
	}

	for _, l := range lines {
		if _, err := exec(ctx, `
			INSERT INTO invoice_lines (id, entity_id, invoice_id, line_no, item_name, item_code, quantity, unit_price, amount, tax_rate, tax_amount, goods_service_code, metadata)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		`, l.ID, l.EntityID, l.InvoiceID, l.LineNo, l.ItemName, nullStr(l.ItemCode), l.Quantity, l.UnitPrice,
			l.Amount, l.TaxRate, l.TaxAmount, nullStr(l.GoodsServiceCode), l.Metadata); err != nil {
			return err
		}
	}
	return nil
}

func (s *InvoiceLineStore) UpsertMany(ctx context.Context, lines []domain.InvoiceLine) error {
	for _, l := range lines {
		if _, err := s.db.Pool.Exec(ctx, `
			INSERT INTO invoice_lines (id, entity_id, invoice_id, line_no, item_name, item_code, quantity, unit_price, amount, tax_rate, tax_amount, goods_service_code, metadata)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
			ON CONFLICT (id) DO UPDATE SET
				line_no = EXCLUDED.line_no,
				item_name = EXCLUDED.item_name,
				item_code = EXCLUDED.item_code,
				quantity = EXCLUDED.quantity,
				unit_price = EXCLUDED.unit_price,
				amount = EXCLUDED.amount,
				tax_rate = EXCLUDED.tax_rate,
				tax_amount = EXCLUDED.tax_amount,
				goods_service_code = EXCLUDED.goods_service_code,
				metadata = EXCLUDED.metadata
		`, l.ID, l.EntityID, l.InvoiceID, l.LineNo, l.ItemName, nullStr(l.ItemCode),
			l.Quantity, l.UnitPrice, l.Amount, l.TaxRate, l.TaxAmount,
			nullStr(l.GoodsServiceCode), l.Metadata); err != nil {
			return err
		}
	}
	return nil
}

func (s *InvoiceLineStore) DeleteByInvoice(ctx context.Context, entityID, invoiceID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		DELETE FROM invoice_lines WHERE entity_id = $1 AND invoice_id = $2
	`, entityID, invoiceID)
	return err
}

func (s *InvoiceLineStore) ListByInvoice(ctx context.Context, entityID, invoiceID string) ([]domain.InvoiceLine, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, entity_id, invoice_id, line_no, item_name, COALESCE(item_code, ''),
		       quantity, unit_price, amount, tax_rate, tax_amount, COALESCE(goods_service_code, ''), metadata
		FROM invoice_lines WHERE entity_id = $1 AND invoice_id = $2 ORDER BY line_no
	`, entityID, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []domain.InvoiceLine
	for rows.Next() {
		var l domain.InvoiceLine
		if err := rows.Scan(&l.ID, &l.EntityID, &l.InvoiceID, &l.LineNo, &l.ItemName, &l.ItemCode,
			&l.Quantity, &l.UnitPrice, &l.Amount, &l.TaxRate, &l.TaxAmount, &l.GoodsServiceCode, &l.Metadata); err != nil {
			return nil, err
		}
		lines = append(lines, l)
	}
	return lines, rows.Err()
}
