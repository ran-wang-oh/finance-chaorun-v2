package postgres

import (
	"context"
	"fmt"

	"finance.chao.run/v2/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type ReconciliationStore struct{ db *DB }

func NewReconciliationStore(db *DB) *ReconciliationStore {
	return &ReconciliationStore{db: db}
}

func (s *ReconciliationStore) UpsertLogistics(ctx context.Context, lr *domain.LogisticsRecord) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO logistics_records (id, entity_id, invoice_id, waybill_no, carrier, status, ship_date, delivery_date, items, notes)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7,''), NULLIF($8,''), $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			waybill_no = EXCLUDED.waybill_no,
			carrier = EXCLUDED.carrier,
			status = EXCLUDED.status,
			ship_date = EXCLUDED.ship_date,
			delivery_date = EXCLUDED.delivery_date,
			items = EXCLUDED.items,
			notes = EXCLUDED.notes,
			updated_at = now()
	`, lr.ID, lr.EntityID, lr.InvoiceID, lr.WaybillNo, lr.Carrier, lr.Status, lr.ShipDate, lr.DeliveryDate, lr.Items, lr.Notes)
	return err
}

func (s *ReconciliationStore) GetLogisticsByInvoice(ctx context.Context, entityID, invoiceID string) (*domain.LogisticsRecord, error) {
	var lr domain.LogisticsRecord
	var shipDate, deliveryDate *string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, entity_id, invoice_id, waybill_no, carrier, status, ship_date, delivery_date, items, notes
		FROM logistics_records WHERE invoice_id = $1 AND entity_id = $2
	`, invoiceID, entityID).Scan(&lr.ID, &lr.EntityID, &lr.InvoiceID, &lr.WaybillNo, &lr.Carrier, &lr.Status, &shipDate, &deliveryDate, &lr.Items, &lr.Notes)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if shipDate != nil {
		lr.ShipDate = *shipDate
	}
	if deliveryDate != nil {
		lr.DeliveryDate = *deliveryDate
	}
	return &lr, nil
}

func (s *ReconciliationStore) DeleteLogistics(ctx context.Context, entityID, invoiceID string) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM logistics_records WHERE invoice_id = $1 AND entity_id = $2`, invoiceID, entityID)
	return err
}

func (s *ReconciliationStore) UpsertBankTransaction(ctx context.Context, bt *domain.BankTransaction) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO bank_transactions (id, entity_id, book_id, transaction_date, counterparty_name, counterparty_account, amount, direction, summary, bank_reference, matched_invoice_id, match_confidence)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NULLIF($11,''), $12)
		ON CONFLICT (id) DO UPDATE SET
			transaction_date = EXCLUDED.transaction_date,
			counterparty_name = EXCLUDED.counterparty_name,
			counterparty_account = EXCLUDED.counterparty_account,
			amount = EXCLUDED.amount,
			direction = EXCLUDED.direction,
			summary = EXCLUDED.summary,
			bank_reference = EXCLUDED.bank_reference,
			matched_invoice_id = EXCLUDED.matched_invoice_id,
			match_confidence = EXCLUDED.match_confidence,
			updated_at = now()
	`, bt.ID, bt.EntityID, bt.BookID, bt.TransactionDate, bt.CounterpartyName, bt.CounterpartyAccount, bt.Amount, bt.Direction, bt.Summary, bt.BankReference, bt.MatchedInvoiceID, bt.MatchConfidence)
	return err
}

func (s *ReconciliationStore) GetBankTransaction(ctx context.Context, entityID, bankTxID string) (*domain.BankTransaction, error) {
	var bt domain.BankTransaction
	var txDate *string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, entity_id, COALESCE(book_id,''), transaction_date, counterparty_name, COALESCE(counterparty_account,''),
		       amount, direction, summary, COALESCE(bank_reference,''), COALESCE(matched_invoice_id,''), match_confidence
		FROM bank_transactions WHERE id = $1 AND entity_id = $2
	`, bankTxID, entityID).Scan(&bt.ID, &bt.EntityID, &bt.BookID, &txDate, &bt.CounterpartyName, &bt.CounterpartyAccount,
		&bt.Amount, &bt.Direction, &bt.Summary, &bt.BankReference, &bt.MatchedInvoiceID, &bt.MatchConfidence)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if txDate != nil {
		bt.TransactionDate = *txDate
	}
	return &bt, nil
}

func (s *ReconciliationStore) MatchBankToInvoice(ctx context.Context, entityID, bankTxID, invoiceID string, confidence float64) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE bank_transactions SET matched_invoice_id = $3, match_confidence = $4, updated_at = now()
		WHERE id = $1 AND entity_id = $2
	`, bankTxID, entityID, invoiceID, confidence)
	return err
}

func (s *ReconciliationStore) UnmatchBankFromInvoice(ctx context.Context, entityID, bankTxID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE bank_transactions SET matched_invoice_id = NULL, match_confidence = 0, updated_at = now()
		WHERE id = $1 AND entity_id = $2
	`, bankTxID, entityID)
	return err
}

func (s *ReconciliationStore) ListUnmatchedBankTransactions(ctx context.Context, entityID, bookID string) ([]domain.BankTransaction, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, entity_id, book_id, transaction_date, counterparty_name, counterparty_account, amount, direction, summary, bank_reference, COALESCE(matched_invoice_id,''), match_confidence
		FROM bank_transactions
		WHERE entity_id = $1 AND matched_invoice_id IS NULL
	`, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.BankTransaction
	for rows.Next() {
		var bt domain.BankTransaction
		var txDate *string
		if err := rows.Scan(&bt.ID, &bt.EntityID, &bt.BookID, &txDate, &bt.CounterpartyName, &bt.CounterpartyAccount, &bt.Amount, &bt.Direction, &bt.Summary, &bt.BankReference, &bt.MatchedInvoiceID, &bt.MatchConfidence); err != nil {
			return nil, err
		}
		if txDate != nil {
			bt.TransactionDate = *txDate
		}
		list = append(list, bt)
	}
	return list, rows.Err()
}

func (s *ReconciliationStore) ThreeWayMatch(ctx context.Context, entityID, bookID, period string) (*domain.ThreeWaySummary, error) {
	query := `
		SELECT
			inv.id AS invoice_id,
			inv.invoice_no,
			inv.seller_name AS counterparty_name,
			inv.total_amount AS amount_with_tax,
			inv.direction,
			COALESCE(lr.status, 'not_applicable') AS logistics_status,
			COALESCE(lr.id, '') AS logistics_id,
			COALESCE(lr.waybill_no, '') AS waybill_no,
			CASE WHEN bt.id IS NOT NULL THEN 'matched' ELSE 'unmatched' END AS bank_status,
			COALESCE(bt.id, '') AS bank_tx_id,
			COALESCE(bt.amount, 0) AS bank_amount
		FROM invoices inv
		JOIN journal_entries je ON je.source_type = 'invoice' AND je.source_id = inv.id
		LEFT JOIN logistics_records lr ON lr.invoice_id = inv.id
		LEFT JOIN bank_transactions bt ON bt.matched_invoice_id = inv.id
		WHERE inv.entity_id = $1
		  AND inv.book_id = $2
		  AND je.period = $3
		  AND inv.status = 'approved'
		ORDER BY inv.invoice_no
	`

	rows, err := s.db.Pool.Query(ctx, query, entityID, bookID, period)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			return nil, fmt.Errorf("three way match: %s (code=%s, hint=%s)", pgErr.Message, pgErr.Code, pgErr.Hint)
		}
		return nil, fmt.Errorf("three way match: %w", err)
	}
	defer rows.Close()

	summary := &domain.ThreeWaySummary{
		Period: period,
		Rows:   []domain.ThreeWayRow{},
	}

	for rows.Next() {
		var r domain.ThreeWayRow
		if err := rows.Scan(&r.InvoiceID, &r.InvoiceNo, &r.CounterpartyName, &r.AmountWithTax, &r.Direction, &r.LogisticsStatus, &r.LogisticsID, &r.WaybillNo, &r.BankStatus, &r.BankTxID, &r.BankAmount); err != nil {
			return nil, err
		}
		summary.Rows = append(summary.Rows, r)
		summary.TotalCount++
		if r.LogisticsStatus != domain.MatchStatusNotApplicable && r.BankStatus == domain.MatchStatusMatched {
			summary.FullMatch++
		}
		if r.LogisticsStatus == domain.MatchStatusNotApplicable {
			summary.MissingLogistics++
		}
		if r.BankStatus == domain.MatchStatusUnmatched {
			summary.MissingBank++
		}
	}
	return summary, rows.Err()
}
