package postgres

import (
	"context"
	"encoding/json"

	"finance.chao.run/v2/internal/domain"

	"github.com/jackc/pgx/v5"
)

type ReportMappingStore struct{ db *DB }

func NewReportMappingStore(db *DB) *ReportMappingStore {
	return &ReportMappingStore{db: db}
}

func (s *ReportMappingStore) ListByReportType(ctx context.Context, reportType, accountingStandard string) ([]domain.ReportMapping, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, report_type, line_code, line_label, display_order, accounting_standard, account_selector, is_subtotal, parent_line_code, formula
		FROM report_mappings
		WHERE report_type = $1 AND accounting_standard = $2
		ORDER BY display_order
	`, reportType, accountingStandard)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReportMappings(rows)
}

func (s *ReportMappingStore) ListAll(ctx context.Context) ([]domain.ReportMapping, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, report_type, line_code, line_label, display_order, accounting_standard, account_selector, is_subtotal, parent_line_code, formula
		FROM report_mappings
		ORDER BY report_type, display_order
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReportMappings(rows)
}

func scanReportMappings(rows pgx.Rows) ([]domain.ReportMapping, error) {
	var result []domain.ReportMapping
	for rows.Next() {
		var m domain.ReportMapping
		var selBytes []byte
		if err := rows.Scan(&m.ID, &m.ReportType, &m.LineCode, &m.LineLabel, &m.DisplayOrder, &m.AccountingStandard, &selBytes, &m.IsSubtotal, &m.ParentLineCode, &m.Formula); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(selBytes, &m.AccountSelector); err != nil {
			m.AccountSelector = domain.AccountSelector{}
		}
		result = append(result, m)
	}
	return result, rows.Err()
}
