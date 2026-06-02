package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"finance.chao.run/v2/internal/domain"

	"github.com/jackc/pgx/v5"
)

type InvoiceStore struct {
	db *DB
}

func NewInvoiceStore(db *DB) *InvoiceStore {
	return &InvoiceStore{db: db}
}

func (s *InvoiceStore) Create(ctx context.Context, inv *domain.Invoice) error {
	extractionJSON, _ := json.Marshal(inv.Extraction)
	evidenceJSON, _ := json.Marshal(inv.EvidenceRefs)
	linesJSON, _ := json.Marshal(inv.InvoiceLines)

	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO invoices (id, entity_id, book_id, invoice_no, invoice_type, direction, issue_date,
			seller_name, seller_tax_no, buyer_name, buyer_tax_no,
			amount, tax_amount, total_amount, currency, status, source,
			invoice_kind, digital_invoice_no, business_tag,
			verification_status, usage_status, deduction_status, red_letter_status,
			original_invoice_id, tax_account_payload,
			extraction, evidence_refs, invoice_lines)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29)
	`, inv.ID, inv.EntityID, inv.BookID, inv.InvoiceNo, inv.InvoiceType, inv.Direction, inv.IssueDate,
		inv.SellerName, inv.SellerTaxNo, inv.BuyerName, inv.BuyerTaxNo,
		inv.AmountWithoutTax, inv.TaxAmount, inv.AmountWithTax, inv.Currency, inv.Status, inv.Source,
		inv.InvoiceKind, inv.DigitalInvoiceNo, inv.BusinessTag,
		inv.VerificationStatus, inv.UsageStatus, inv.DeductionStatus, inv.RedLetterStatus,
		nullStr(inv.OriginalInvoiceID), inv.TaxAccountPayload,
		extractionJSON, evidenceJSON, linesJSON)
	return err
}

func (s *InvoiceStore) Get(ctx context.Context, entityID, invoiceID string) (*domain.Invoice, error) {
	row := s.db.Pool.QueryRow(ctx, `
		SELECT id, entity_id, book_id, invoice_no, invoice_type, direction,
		       to_char(issue_date, 'YYYY-MM-DD'),
		       COALESCE(seller_name, ''), COALESCE(seller_tax_no, ''), COALESCE(buyer_name, ''), COALESCE(buyer_tax_no, ''),
		       amount, tax_amount, total_amount, currency, status, source,
		       COALESCE(invoice_kind, ''), COALESCE(digital_invoice_no, ''), COALESCE(business_tag, ''),
		       COALESCE(verification_status, ''), COALESCE(usage_status, ''), COALESCE(deduction_status, ''), COALESCE(red_letter_status, ''),
		       COALESCE(original_invoice_id, ''), COALESCE(tax_account_payload, '{}'),
		       extraction, evidence_refs, invoice_lines,
		       to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM invoices WHERE id = $1 AND entity_id = $2
	`, invoiceID, entityID)

	return scanInvoice(row)
}

func (s *InvoiceStore) List(ctx context.Context, entityID, bookID, period, status string, limit, offset int) ([]domain.Invoice, error) {
	query := `
		SELECT id, entity_id, book_id, invoice_no, invoice_type, direction,
		       to_char(issue_date, 'YYYY-MM-DD'),
		       COALESCE(seller_name, ''), COALESCE(seller_tax_no, ''), COALESCE(buyer_name, ''), COALESCE(buyer_tax_no, ''),
		       amount, tax_amount, total_amount, currency, status, source,
		       COALESCE(invoice_kind, ''), COALESCE(digital_invoice_no, ''), COALESCE(business_tag, ''),
		       COALESCE(verification_status, ''), COALESCE(usage_status, ''), COALESCE(deduction_status, ''), COALESCE(red_letter_status, ''),
		       COALESCE(original_invoice_id, ''), COALESCE(tax_account_payload, '{}'),
		       extraction, evidence_refs, invoice_lines,
		       to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM invoices WHERE entity_id = $1
	`
	args := []any{entityID}
	argIdx := 2

	if bookID != "" {
		query += fmt.Sprintf(` AND book_id = $%d`, argIdx)
		args = append(args, bookID)
		argIdx++
	}
	if period != "" {
		query += fmt.Sprintf(` AND to_char(issue_date, 'YYYY-MM') = $%d`, argIdx)
		args = append(args, period)
		argIdx++
	}
	if status != "" {
		query += fmt.Sprintf(` AND status = $%d`, argIdx)
		args = append(args, status)
		argIdx++
	}

	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []domain.Invoice
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, err
		}
		invoices = append(invoices, *inv)
	}
	return invoices, rows.Err()
}

func (s *InvoiceStore) FindByInvoiceNo(ctx context.Context, entityID, bookID, invoiceNo string) (*domain.Invoice, error) {
	row := s.db.Pool.QueryRow(ctx, `
		SELECT id, entity_id, book_id, invoice_no, invoice_type, direction,
		       to_char(issue_date, 'YYYY-MM-DD'),
		       COALESCE(seller_name, ''), COALESCE(seller_tax_no, ''), COALESCE(buyer_name, ''), COALESCE(buyer_tax_no, ''),
		       amount, tax_amount, total_amount, currency, status, source,
		       COALESCE(invoice_kind, ''), COALESCE(digital_invoice_no, ''), COALESCE(business_tag, ''),
		       COALESCE(verification_status, ''), COALESCE(usage_status, ''), COALESCE(deduction_status, ''), COALESCE(red_letter_status, ''),
		       COALESCE(original_invoice_id, ''), COALESCE(tax_account_payload, '{}'),
		       extraction, evidence_refs, invoice_lines,
		       to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM invoices WHERE entity_id = $1 AND book_id = $2 AND invoice_no = $3
	`, entityID, bookID, invoiceNo)

	return scanInvoice(row)
}

func (s *InvoiceStore) Approve(ctx context.Context, entityID, invoiceID string) (*domain.Invoice, error) {
	return s.updateStatus(ctx, entityID, invoiceID, domain.StatusPendingReview, domain.StatusApproved)
}

func (s *InvoiceStore) ApproveTx(ctx context.Context, tx pgx.Tx, entityID, invoiceID string) (*domain.Invoice, error) {
	return s.updateStatusTx(ctx, tx, entityID, invoiceID, domain.StatusPendingReview, domain.StatusApproved)
}

func (s *InvoiceStore) Reject(ctx context.Context, entityID, invoiceID string) (*domain.Invoice, error) {
	return s.updateStatus(ctx, entityID, invoiceID, domain.StatusPendingReview, domain.StatusRejected)
}

func (s *InvoiceStore) MarkPosted(ctx context.Context, entityID, invoiceID string) (*domain.Invoice, error) {
	return s.updateStatus(ctx, entityID, invoiceID, domain.StatusApproved, domain.StatusPosted)
}

func (s *InvoiceStore) MarkPostedTx(ctx context.Context, tx pgx.Tx, entityID, invoiceID string) (*domain.Invoice, error) {
	return s.updateStatusTx(ctx, tx, entityID, invoiceID, domain.StatusApproved, domain.StatusPosted)
}

func (s *InvoiceStore) ListPostedByPeriod(ctx context.Context, entityID, bookID, period string) ([]domain.Invoice, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, entity_id, book_id, invoice_no, invoice_type, direction,
		       to_char(issue_date, 'YYYY-MM-DD'),
		       COALESCE(seller_name, ''), COALESCE(seller_tax_no, ''), COALESCE(buyer_name, ''), COALESCE(buyer_tax_no, ''),
		       amount, tax_amount, total_amount, currency, status, source,
		       COALESCE(invoice_kind, ''), COALESCE(digital_invoice_no, ''), COALESCE(business_tag, ''),
		       COALESCE(verification_status, ''), COALESCE(usage_status, ''), COALESCE(deduction_status, ''), COALESCE(red_letter_status, ''),
		       COALESCE(original_invoice_id, ''), COALESCE(tax_account_payload, '{}'),
		       extraction, evidence_refs, invoice_lines,
		       to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM invoices WHERE entity_id = $1 AND book_id = $2 AND to_char(issue_date, 'YYYY-MM') = $3 AND status = 'posted'
		ORDER BY issue_date
	`, entityID, bookID, period)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []domain.Invoice
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, err
		}
		invoices = append(invoices, *inv)
	}
	return invoices, rows.Err()
}

func (s *InvoiceStore) Update(ctx context.Context, inv *domain.Invoice) error {
	extractionJSON, _ := json.Marshal(inv.Extraction)
	evidenceJSON, _ := json.Marshal(inv.EvidenceRefs)
	linesJSON, _ := json.Marshal(inv.InvoiceLines)

	_, err := s.db.Pool.Exec(ctx, `
		UPDATE invoices SET
			invoice_no = $1, invoice_type = $2, direction = $3, issue_date = $4,
			seller_name = $5, seller_tax_no = $6, buyer_name = $7, buyer_tax_no = $8,
			amount = $9, tax_amount = $10, total_amount = $11, currency = $12,
			invoice_kind = $13, digital_invoice_no = $14, business_tag = $15,
			verification_status = $16, usage_status = $17, deduction_status = $18,
			red_letter_status = $19, original_invoice_id = NULLIF($20, ''),
			tax_account_payload = $21, extraction = $22, evidence_refs = $23,
			invoice_lines = $24, updated_at = now()
		WHERE id = $25 AND entity_id = $26
	`, inv.InvoiceNo, inv.InvoiceType, inv.Direction, inv.IssueDate,
		inv.SellerName, inv.SellerTaxNo, inv.BuyerName, inv.BuyerTaxNo,
		inv.AmountWithoutTax, inv.TaxAmount, inv.AmountWithTax, inv.Currency,
		inv.InvoiceKind, inv.DigitalInvoiceNo, inv.BusinessTag,
		inv.VerificationStatus, inv.UsageStatus, inv.DeductionStatus,
		inv.RedLetterStatus, nullStr(inv.OriginalInvoiceID),
		inv.TaxAccountPayload, extractionJSON, evidenceJSON, linesJSON,
		inv.ID, inv.EntityID)
	return err
}

func (s *InvoiceStore) FindByDigitalInvoiceNo(ctx context.Context, entityID, bookID, digitalNo string) (*domain.Invoice, error) {
	row := s.db.Pool.QueryRow(ctx, `
		SELECT id, entity_id, book_id, invoice_no, invoice_type, direction,
		       to_char(issue_date, 'YYYY-MM-DD'),
		       COALESCE(seller_name, ''), COALESCE(seller_tax_no, ''), COALESCE(buyer_name, ''), COALESCE(buyer_tax_no, ''),
		       amount, tax_amount, total_amount, currency, status, source,
		       COALESCE(invoice_kind, ''), COALESCE(digital_invoice_no, ''), COALESCE(business_tag, ''),
		       COALESCE(verification_status, ''), COALESCE(usage_status, ''), COALESCE(deduction_status, ''), COALESCE(red_letter_status, ''),
		       COALESCE(original_invoice_id, ''), COALESCE(tax_account_payload, '{}'),
		       extraction, evidence_refs, invoice_lines,
		       to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM invoices WHERE entity_id = $1 AND book_id = $2 AND digital_invoice_no = $3
	`, entityID, bookID, digitalNo)
	return scanInvoice(row)
}

func (s *InvoiceStore) FindByOriginalInvoiceID(ctx context.Context, entityID, originalInvoiceID string) ([]domain.Invoice, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, entity_id, book_id, invoice_no, invoice_type, direction,
		       to_char(issue_date, 'YYYY-MM-DD'),
		       COALESCE(seller_name, ''), COALESCE(seller_tax_no, ''), COALESCE(buyer_name, ''), COALESCE(buyer_tax_no, ''),
		       amount, tax_amount, total_amount, currency, status, source,
		       COALESCE(invoice_kind, ''), COALESCE(digital_invoice_no, ''), COALESCE(business_tag, ''),
		       COALESCE(verification_status, ''), COALESCE(usage_status, ''), COALESCE(deduction_status, ''), COALESCE(red_letter_status, ''),
		       COALESCE(original_invoice_id, ''), COALESCE(tax_account_payload, '{}'),
		       extraction, evidence_refs, invoice_lines,
		       to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM invoices WHERE entity_id = $1 AND original_invoice_id = $2
		ORDER BY created_at
	`, entityID, originalInvoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []domain.Invoice
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, err
		}
		invoices = append(invoices, *inv)
	}
	return invoices, rows.Err()
}

func (s *InvoiceStore) UpdateVerificationStatus(ctx context.Context, entityID, invoiceID, status string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE invoices SET verification_status = $3, updated_at = now()
		WHERE id = $1 AND entity_id = $2
	`, invoiceID, entityID, status)
	return err
}

func (s *InvoiceStore) UpdateUsageStatus(ctx context.Context, entityID, invoiceID, status string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE invoices SET usage_status = $3, updated_at = now()
		WHERE id = $1 AND entity_id = $2
	`, invoiceID, entityID, status)
	return err
}

func (s *InvoiceStore) UpdateDeductionStatus(ctx context.Context, entityID, invoiceID, status string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE invoices SET deduction_status = $3, updated_at = now()
		WHERE id = $1 AND entity_id = $2
	`, invoiceID, entityID, status)
	return err
}

func (s *InvoiceStore) UpdateRedLetterStatus(ctx context.Context, entityID, invoiceID, status string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE invoices SET red_letter_status = $3, updated_at = now()
		WHERE id = $1 AND entity_id = $2
	`, invoiceID, entityID, status)
	return err
}

func (s *InvoiceStore) updateStatus(ctx context.Context, entityID, invoiceID, fromStatus, toStatus string) (*domain.Invoice, error) {
	tag, err := s.db.Pool.Exec(ctx, `
		UPDATE invoices SET status = $3, updated_at = now() WHERE id = $1 AND entity_id = $2 AND status = $4
	`, invoiceID, entityID, toStatus, fromStatus)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("invoice not found or wrong status: %s", invoiceID)
	}
	return s.Get(ctx, entityID, invoiceID)
}

func (s *InvoiceStore) updateStatusTx(ctx context.Context, tx pgx.Tx, entityID, invoiceID, fromStatus, toStatus string) (*domain.Invoice, error) {
	tag, err := tx.Exec(ctx, `
		UPDATE invoices SET status = $3, updated_at = now() WHERE id = $1 AND entity_id = $2 AND status = $4
	`, invoiceID, entityID, toStatus, fromStatus)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("invoice not found or wrong status: %s", invoiceID)
	}
	return s.Get(ctx, entityID, invoiceID)
}

func scanInvoice(row pgx.Row) (*domain.Invoice, error) {
	var inv domain.Invoice
	var extractionRaw, evidenceRaw, linesRaw []byte

	err := row.Scan(&inv.ID, &inv.EntityID, &inv.BookID, &inv.InvoiceNo, &inv.InvoiceType, &inv.Direction,
		&inv.IssueDate, &inv.SellerName, &inv.SellerTaxNo, &inv.BuyerName, &inv.BuyerTaxNo,
		&inv.AmountWithoutTax, &inv.TaxAmount, &inv.AmountWithTax, &inv.Currency, &inv.Status, &inv.Source,
		&inv.InvoiceKind, &inv.DigitalInvoiceNo, &inv.BusinessTag,
		&inv.VerificationStatus, &inv.UsageStatus, &inv.DeductionStatus, &inv.RedLetterStatus,
		&inv.OriginalInvoiceID, &inv.TaxAccountPayload,
		&extractionRaw, &evidenceRaw, &linesRaw,
		&inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	inv.Extraction = extractionRaw
	inv.EvidenceRefs = evidenceRaw
	if len(linesRaw) > 0 {
		json.Unmarshal(linesRaw, &inv.InvoiceLines)
	}

	return &inv, nil
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
