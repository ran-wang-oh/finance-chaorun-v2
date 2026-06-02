package postgres

import (
	"context"
	"fmt"

	"finance.chao.run/v2/internal/domain"

	"github.com/jackc/pgx/v5"
)

type AccountStore struct {
	db *DB
}

func NewAccountStore(db *DB) *AccountStore {
	return &AccountStore{db: db}
}

func (s *AccountStore) Create(ctx context.Context, a *domain.ChartAccount) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO chart_of_accounts (id, entity_id, book_id, code, name, category, balance_type, parent_id, is_system, tax_relevant, keywords)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`, a.ID, a.EntityID, a.BookID, a.Code, a.Name, a.Category, a.BalanceType,
		nullStr(a.ParentID), a.IsSystem, a.TaxRelevant, a.Keywords)
	return err
}

func (s *AccountStore) GetByCode(ctx context.Context, entityID, bookID, code string) (*domain.ChartAccount, error) {
	row := s.db.Pool.QueryRow(ctx, `
		SELECT id, entity_id, book_id, code, name, category, balance_type,
		       COALESCE(parent_id, ''), is_system, tax_relevant, keywords,
		       to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM chart_of_accounts WHERE entity_id = $1 AND book_id = $2 AND code = $3
	`, entityID, bookID, code)

	var a domain.ChartAccount
	err := row.Scan(&a.ID, &a.EntityID, &a.BookID, &a.Code, &a.Name, &a.Category, &a.BalanceType,
		&a.ParentID, &a.IsSystem, &a.TaxRelevant, &a.Keywords, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (s *AccountStore) List(ctx context.Context, query domain.AccountListQuery) ([]domain.ChartAccount, int, error) {
	baseQuery := ` FROM chart_of_accounts WHERE entity_id = $1`
	args := []any{query.EntityID}
	argIdx := 2

	if query.Category != "" {
		baseQuery += fmt.Sprintf(` AND category = $%d`, argIdx)
		args = append(args, query.Category)
		argIdx++
	}
	if query.Keyword != "" {
		baseQuery += fmt.Sprintf(` AND (name ILIKE $%d OR code ILIKE $%d)`, argIdx, argIdx)
		args = append(args, "%"+query.Keyword+"%")
		argIdx++
	}

	var total int
	if err := s.db.Pool.QueryRow(ctx, `SELECT count(*)`+baseQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	selectQuery := `SELECT id, entity_id, book_id, code, name, category, balance_type,
		COALESCE(parent_id, ''), is_system, tax_relevant, keywords,
		to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')` + baseQuery

	selectQuery += fmt.Sprintf(` ORDER BY code LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
	args = append(args, query.Limit, query.Offset)

	rows, err := s.db.Pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var accounts []domain.ChartAccount
	for rows.Next() {
		var a domain.ChartAccount
		if err := rows.Scan(&a.ID, &a.EntityID, &a.BookID, &a.Code, &a.Name, &a.Category, &a.BalanceType,
			&a.ParentID, &a.IsSystem, &a.TaxRelevant, &a.Keywords, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, 0, err
		}
		accounts = append(accounts, a)
	}
	return accounts, total, rows.Err()
}

func (s *AccountStore) Update(ctx context.Context, a *domain.ChartAccount) error {
	tag, err := s.db.Pool.Exec(ctx, `
		UPDATE chart_of_accounts SET name = $4, category = $5, balance_type = $6, keywords = $7, tax_relevant = $8, updated_at = now()
		WHERE id = $1 AND entity_id = $2 AND code = $3
	`, a.ID, a.EntityID, a.Code, a.Name, a.Category, a.BalanceType, a.Keywords, a.TaxRelevant)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("account not found: %s", a.Code)
	}
	return nil
}

func (s *AccountStore) Delete(ctx context.Context, entityID, accountID string) error {
	tag, err := s.db.Pool.Exec(ctx, `
		DELETE FROM chart_of_accounts WHERE id = $1 AND entity_id = $2
	`, accountID, entityID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("account not found: %s", accountID)
	}
	return nil
}
