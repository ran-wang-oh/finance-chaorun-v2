package postgres

import (
	"context"
	"time"

	"finance.chao.run/v2/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PeriodStore struct {
	db *DB
}

func NewPeriodStore(db *DB) *PeriodStore {
	return &PeriodStore{db: db}
}

func (s *PeriodStore) GetOrCreate(ctx context.Context, p *domain.AccountingPeriod) (*domain.AccountingPeriod, error) {
	existing, err := s.Get(ctx, p.EntityID, p.BookID, p.Period)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	p.ID = uuid.NewString()
	p.Status = domain.PeriodStatusOpen
	p.OpenedAt = time.Now().UTC().Format(time.RFC3339)

	_, err = s.db.Pool.Exec(ctx, `
		INSERT INTO accounting_periods (id, entity_id, book_id, period, status, opened_at)
		VALUES ($1, $2, $3, $4, $5, now())
	`, p.ID, p.EntityID, p.BookID, p.Period, p.Status)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *PeriodStore) Get(ctx context.Context, entityID, bookID, period string) (*domain.AccountingPeriod, error) {
	row := s.db.Pool.QueryRow(ctx, `
		SELECT id, entity_id, book_id, period, status,
		       COALESCE(to_char(opened_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
		       COALESCE(to_char(closing_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
		       COALESCE(to_char(closed_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
		       COALESCE(to_char(locked_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
		       COALESCE(closed_by, '')
		FROM accounting_periods WHERE entity_id = $1 AND book_id = $2 AND period = $3
	`, entityID, bookID, period)

	var p domain.AccountingPeriod
	err := row.Scan(&p.ID, &p.EntityID, &p.BookID, &p.Period, &p.Status,
		&p.OpenedAt, &p.ClosingAt, &p.ClosedAt, &p.LockedAt, &p.ClosedBy)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (s *PeriodStore) ListByBook(ctx context.Context, entityID, bookID string) ([]domain.AccountingPeriod, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, entity_id, book_id, period, status,
		       COALESCE(to_char(opened_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
		       COALESCE(to_char(closing_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
		       COALESCE(to_char(closed_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
		       COALESCE(to_char(locked_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
		       COALESCE(closed_by, '')
		FROM accounting_periods WHERE entity_id = $1 AND book_id = $2 ORDER BY period
	`, entityID, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var periods []domain.AccountingPeriod
	for rows.Next() {
		var p domain.AccountingPeriod
		if err := rows.Scan(&p.ID, &p.EntityID, &p.BookID, &p.Period, &p.Status,
			&p.OpenedAt, &p.ClosingAt, &p.ClosedAt, &p.LockedAt, &p.ClosedBy); err != nil {
			return nil, err
		}
		periods = append(periods, p)
	}
	return periods, rows.Err()
}

func (s *PeriodStore) IsClosed(ctx context.Context, entityID, bookID, period string) (bool, error) {
	var status string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT status FROM accounting_periods WHERE entity_id = $1 AND book_id = $2 AND period = $3
	`, entityID, bookID, period).Scan(&status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return status == domain.PeriodStatusClosed || status == domain.PeriodStatusLocked, nil
}

func (s *PeriodStore) UpdateStatus(ctx context.Context, entityID, bookID, period, fromStatus, toStatus string, closedBy *string) (*domain.AccountingPeriod, error) {
	var timestampCol string
	switch toStatus {
	case domain.PeriodStatusClosing:
		timestampCol = "closing_at = now()"
	case domain.PeriodStatusClosed:
		timestampCol = "closed_at = now()"
	case domain.PeriodStatusLocked:
		timestampCol = "locked_at = now()"
	default:
		timestampCol = "opened_at = now()"
	}

	query := `UPDATE accounting_periods SET status = $4, ` + timestampCol
	args := []any{entityID, bookID, period, toStatus}
	argIdx := 5

	if closedBy != nil {
		query += `, closed_by = $` + string(rune('0'+argIdx))
		args = append(args, *closedBy)
		argIdx++
	}
	query += ` WHERE entity_id = $1 AND book_id = $2 AND period = $3 AND status = $` + string(rune('0'+argIdx))
	args = append(args, fromStatus)

	tag, err := s.db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}

	p, err := s.Get(ctx, entityID, bookID, period)
	if err != nil {
		return nil, err
	}
	return p, nil
}
