package postgres

import (
	"context"

	"finance.chao.run/v2/internal/domain"
)

// ---- AdjustmentStore ----

type AdjustmentStore struct{ db *DB }

func NewAdjustmentStore(db *DB) *AdjustmentStore { return &AdjustmentStore{db: db} }

func (s *AdjustmentStore) Upsert(ctx context.Context, rec *domain.AdjustmentRecord) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO tax_adjustments (id, entity_id, book_id, tax_year, category, book_amount, tax_base, adjustment, formula, detail)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (entity_id, tax_year, category, id) DO UPDATE SET
			book_amount = EXCLUDED.book_amount,
			tax_base = EXCLUDED.tax_base,
			adjustment = EXCLUDED.adjustment,
			formula = EXCLUDED.formula,
			detail = EXCLUDED.detail
	`, rec.ID, rec.EntityID, rec.BookID, rec.TaxYear, rec.Category, rec.BookAmount, rec.TaxBase, rec.Adjustment, rec.Formula, rec.Detail)
	return err
}

func (s *AdjustmentStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM tax_adjustments WHERE id = $1`, id)
	return err
}

func (s *AdjustmentStore) List(ctx context.Context, query domain.AdjustmentListQuery) ([]domain.AdjustmentRecord, int, error) {
	if query.Limit <= 0 {
		query.Limit = 50
	}
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, entity_id, book_id, tax_year, category, book_amount, tax_base, adjustment, formula, detail
		FROM tax_adjustments
		WHERE entity_id = $1 AND tax_year = $2 AND ($3 = '' OR category = $3)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5
	`, query.EntityID, query.TaxYear, query.Category, query.Limit, query.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []domain.AdjustmentRecord
	for rows.Next() {
		var r domain.AdjustmentRecord
		if err := rows.Scan(&r.ID, &r.EntityID, &r.BookID, &r.TaxYear, &r.Category, &r.BookAmount, &r.TaxBase, &r.Adjustment, &r.Formula, &r.Detail); err != nil {
			return nil, 0, err
		}
		items = append(items, r)
	}
	// Total count approximation — just return items length for now
	return items, len(items), rows.Err()
}

func (s *AdjustmentStore) DeleteByCategory(ctx context.Context, entityID, taxYear, category string) error {
	_, err := s.db.Pool.Exec(ctx, `
		DELETE FROM tax_adjustments WHERE entity_id = $1 AND tax_year = $2 AND category = $3
	`, entityID, taxYear, category)
	return err
}

func (s *AdjustmentStore) SumByYear(ctx context.Context, entityID, taxYear string) (float64, error) {
	var total float64
	err := s.db.Pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(adjustment), 0) FROM tax_adjustments WHERE entity_id = $1 AND tax_year = $2
	`, entityID, taxYear).Scan(&total)
	return total, err
}

// ---- TaxProfileStore ----

type TaxProfileStore struct{ db *DB }

func NewTaxProfileStore(db *DB) *TaxProfileStore { return &TaxProfileStore{db: db} }

func (s *TaxProfileStore) Upsert(ctx context.Context, p *domain.TaxProfile) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO tax_profiles (id, entity_id, book_id, taxpayer_type, vat_taxpayer_type, cit_rate_type, tax_registration_no, tax_office, industry)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (entity_id, book_id) DO UPDATE SET
			taxpayer_type = EXCLUDED.taxpayer_type,
			vat_taxpayer_type = EXCLUDED.vat_taxpayer_type,
			cit_rate_type = EXCLUDED.cit_rate_type,
			tax_registration_no = EXCLUDED.tax_registration_no,
			tax_office = EXCLUDED.tax_office,
			industry = EXCLUDED.industry,
			updated_at = now()
	`, p.ID, p.EntityID, p.BookID, p.TaxpayerType, p.VATTaxpayerType, p.CITRateType, p.TaxRegistrationNo, p.TaxOffice, p.Industry)
	return err
}

func (s *TaxProfileStore) Get(ctx context.Context, entityID, bookID string) (*domain.TaxProfile, error) {
	var p domain.TaxProfile
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, entity_id, book_id, taxpayer_type, vat_taxpayer_type, cit_rate_type, tax_registration_no, tax_office, industry
		FROM tax_profiles WHERE entity_id = $1 AND book_id = $2
	`, entityID, bookID).Scan(&p.ID, &p.EntityID, &p.BookID, &p.TaxpayerType, &p.VATTaxpayerType, &p.CITRateType, &p.TaxRegistrationNo, &p.TaxOffice, &p.Industry)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ---- TaxReturnStore ----

type TaxReturnStore struct{ db *DB }

func NewTaxReturnStore(db *DB) *TaxReturnStore { return &TaxReturnStore{db: db} }

func (s *TaxReturnStore) Upsert(ctx context.Context, r *domain.TaxReturn) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO tax_returns (id, entity_id, book_id, tax_year, tax_period, return_type, payload, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (entity_id, book_id, tax_year, tax_period, return_type) DO UPDATE SET
			payload = EXCLUDED.payload,
			status = EXCLUDED.status
	`, r.ID, r.EntityID, r.BookID, r.TaxYear, r.TaxPeriod, r.ReturnType, r.Payload, r.Status)
	return err
}

func (s *TaxReturnStore) Get(ctx context.Context, entityID, bookID, taxYear, taxPeriod, returnType string) (*domain.TaxReturn, error) {
	var r domain.TaxReturn
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, entity_id, book_id, tax_year, tax_period, return_type, payload, status
		FROM tax_returns WHERE entity_id = $1 AND book_id = $2 AND tax_year = $3 AND tax_period = $4 AND return_type = $5
	`, entityID, bookID, taxYear, taxPeriod, returnType).Scan(&r.ID, &r.EntityID, &r.BookID, &r.TaxYear, &r.TaxPeriod, &r.ReturnType, &r.Payload, &r.Status)
	if err != nil {
		return nil, err
	}
	return &r, nil
}
