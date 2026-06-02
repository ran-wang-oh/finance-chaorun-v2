package postgres

import (
	"context"
	"fmt"

	"finance.chao.run/v2/internal/domain"

	"github.com/jackc/pgx/v5"
)

type BookStore struct {
	db *DB
}

func NewBookStore(db *DB) *BookStore {
	return &BookStore{db: db}
}

func (s *BookStore) Create(ctx context.Context, book *domain.AccountingBook) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO accounting_books (id, entity_id, code, name, accounting_standard, base_currency, start_period, is_default, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, book.ID, book.EntityID, book.Code, book.Name, book.AccountingStandard, book.BaseCurrency, book.StartPeriod, book.IsDefault, book.Status)
	return err
}

func (s *BookStore) GetByID(ctx context.Context, entityID, bookID string) (*domain.AccountingBook, error) {
	row := s.db.Pool.QueryRow(ctx, `
		SELECT id, entity_id, code, name, accounting_standard, base_currency, start_period, is_default, status,
		       to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM accounting_books WHERE id = $1 AND entity_id = $2
	`, bookID, entityID)

	var b domain.AccountingBook
	err := row.Scan(&b.ID, &b.EntityID, &b.Code, &b.Name, &b.AccountingStandard, &b.BaseCurrency,
		&b.StartPeriod, &b.IsDefault, &b.Status, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &b, nil
}

func (s *BookStore) GetDefault(ctx context.Context, entityID string) (*domain.AccountingBook, error) {
	row := s.db.Pool.QueryRow(ctx, `
		SELECT id, entity_id, code, name, accounting_standard, base_currency, start_period, is_default, status,
		       to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM accounting_books WHERE entity_id = $1 AND is_default = true LIMIT 1
	`, entityID)

	var b domain.AccountingBook
	err := row.Scan(&b.ID, &b.EntityID, &b.Code, &b.Name, &b.AccountingStandard, &b.BaseCurrency,
		&b.StartPeriod, &b.IsDefault, &b.Status, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &b, nil
}

func (s *BookStore) List(ctx context.Context, entityID string) ([]domain.AccountingBook, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, entity_id, code, name, accounting_standard, base_currency, start_period, is_default, status,
		       to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM accounting_books WHERE entity_id = $1 ORDER BY code
	`, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []domain.AccountingBook
	for rows.Next() {
		var b domain.AccountingBook
		if err := rows.Scan(&b.ID, &b.EntityID, &b.Code, &b.Name, &b.AccountingStandard, &b.BaseCurrency,
			&b.StartPeriod, &b.IsDefault, &b.Status, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	return books, rows.Err()
}

func (s *BookStore) Update(ctx context.Context, book *domain.AccountingBook) error {
	tag, err := s.db.Pool.Exec(ctx, `
		UPDATE accounting_books SET name = $3, accounting_standard = $4, base_currency = $5, status = $6, updated_at = now()
		WHERE id = $1 AND entity_id = $2
	`, book.ID, book.EntityID, book.Name, book.AccountingStandard, book.BaseCurrency, book.Status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("book not found: %s", book.ID)
	}
	return nil
}

func (s *BookStore) SetDefault(ctx context.Context, entityID, bookID string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE accounting_books SET is_default = false WHERE entity_id = $1`, entityID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE accounting_books SET is_default = true WHERE id = $1 AND entity_id = $2`, bookID, entityID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("book not found: %s", bookID)
	}
	return tx.Commit(ctx)
}
