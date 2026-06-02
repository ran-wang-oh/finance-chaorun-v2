package postgres

import (
	"context"

	"finance.chao.run/v2/internal/domain"

	"github.com/jackc/pgx/v5"
)

// ---- RiskScanStore ----

type RiskScanStore struct{ db *DB }

func NewRiskScanStore(db *DB) *RiskScanStore { return &RiskScanStore{db: db} }

func (s *RiskScanStore) Create(ctx context.Context, scan *domain.RiskScan) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO risk_scans (id, entity_id, book_id, period, scan_type, rules_triggered, findings, total_score, risk_level, engine_version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, scan.ID, scan.EntityID, scan.BookID, scan.Period, scan.ScanType, scan.RulesTriggered, scan.Findings, scan.TotalScore, scan.RiskLevel, scan.EngineVersion)
	return err
}

func (s *RiskScanStore) ListByPeriod(ctx context.Context, entityID, bookID, period string) ([]domain.RiskScan, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, entity_id, book_id, period, scan_type, rules_triggered, findings, total_score, risk_level, COALESCE(engine_version,'')
		FROM risk_scans
		WHERE entity_id = $1 AND book_id = $2 AND period = $3
		ORDER BY created_at DESC
	`, entityID, bookID, period)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scans []domain.RiskScan
	for rows.Next() {
		var s domain.RiskScan
		if err := rows.Scan(&s.ID, &s.EntityID, &s.BookID, &s.Period, &s.ScanType, &s.RulesTriggered, &s.Findings, &s.TotalScore, &s.RiskLevel, &s.EngineVersion); err != nil {
			return nil, err
		}
		scans = append(scans, s)
	}
	return scans, rows.Err()
}

func (s *RiskScanStore) GetLatest(ctx context.Context, entityID, bookID, period string) (*domain.RiskScan, error) {
	var scan domain.RiskScan
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, entity_id, book_id, period, scan_type, rules_triggered, findings, total_score, risk_level, COALESCE(engine_version,'')
		FROM risk_scans
		WHERE entity_id = $1 AND book_id = $2 AND period = $3
		ORDER BY created_at DESC LIMIT 1
	`, entityID, bookID, period).Scan(&scan.ID, &scan.EntityID, &scan.BookID, &scan.Period, &scan.ScanType, &scan.RulesTriggered, &scan.Findings, &scan.TotalScore, &scan.RiskLevel, &scan.EngineVersion)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &scan, nil
}

// ---- ConsistencyCheckStore ----

type ConsistencyCheckStore struct{ db *DB }

func NewConsistencyCheckStore(db *DB) *ConsistencyCheckStore {
	return &ConsistencyCheckStore{db: db}
}

func (s *ConsistencyCheckStore) CreateMany(ctx context.Context, checks []domain.ConsistencyCheck) error {
	for _, c := range checks {
		_, err := s.db.Pool.Exec(ctx, `
			INSERT INTO consistency_checks (id, entity_id, book_id, period, check_type, check_name, source_value, target_value, difference, tolerance, passed, detail)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`, c.ID, c.EntityID, c.BookID, c.Period, c.CheckType, c.CheckName, c.SourceValue, c.TargetValue, c.Difference, c.Tolerance, c.Passed, c.Detail)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ConsistencyCheckStore) ListByPeriod(ctx context.Context, entityID, bookID, period string) ([]domain.ConsistencyCheck, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, entity_id, book_id, period, check_type, check_name, source_value, target_value, difference, tolerance, passed, detail
		FROM consistency_checks
		WHERE entity_id = $1 AND book_id = $2 AND period = $3
		ORDER BY created_at DESC
	`, entityID, bookID, period)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []domain.ConsistencyCheck
	for rows.Next() {
		var c domain.ConsistencyCheck
		if err := rows.Scan(&c.ID, &c.EntityID, &c.BookID, &c.Period, &c.CheckType, &c.CheckName, &c.SourceValue, &c.TargetValue, &c.Difference, &c.Tolerance, &c.Passed, &c.Detail); err != nil {
			return nil, err
		}
		checks = append(checks, c)
	}
	return checks, rows.Err()
}
