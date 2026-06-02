package postgres

import (
	"context"
	"fmt"

	"finance.chao.run/v2/internal/domain"

	"github.com/jackc/pgx/v5"
)

type JournalStore struct {
	db *DB
}

func NewJournalStore(db *DB) *JournalStore {
	return &JournalStore{db: db}
}

func (s *JournalStore) Create(ctx context.Context, entry *domain.JournalEntry) error {
	return s.CreateTx(ctx, nil, entry)
}

func (s *JournalStore) CreateTx(ctx context.Context, tx pgx.Tx, entry *domain.JournalEntry) error {
	exec := s.db.Pool.Exec
	if tx != nil {
		exec = tx.Exec
	}

	if _, err := exec(ctx, `
		INSERT INTO journal_entries (id, entity_id, book_id, period, voucher_no, voucher_word, entry_date, summary, source_type, source_id, status, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`, entry.ID, entry.EntityID, entry.BookID, entry.Period, entry.VoucherNo, entry.VoucherWord,
		entry.EntryDate, entry.Summary, nullStr(entry.SourceType), nullStr(entry.SourceID),
		entry.Status, nullStr(entry.CreatedBy)); err != nil {
		return err
	}

	for i := range entry.Lines {
		l := &entry.Lines[i]
		l.EntityID = entry.EntityID
		l.JournalEntryID = entry.ID

		aux := l.Auxiliary
		if len(aux) == 0 {
			aux = []byte("{}")
		}
		if _, err := exec(ctx, `
			INSERT INTO journal_lines (id, entity_id, journal_entry_id, account_id, account_code, account_name, direction, debit_amount, credit_amount, currency, line_no, auxiliary)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		`, l.ID, l.EntityID, l.JournalEntryID, l.AccountID, l.AccountCode, l.AccountName,
			l.Direction, l.DebitAmount, l.CreditAmount, l.Currency, l.LineNo, aux); err != nil {
			return err
		}
	}
	return nil
}

func (s *JournalStore) Get(ctx context.Context, entityID, journalID string) (*domain.JournalEntry, error) {
	row := s.db.Pool.QueryRow(ctx, `
		SELECT id, entity_id, book_id, period, COALESCE(voucher_no, ''), voucher_word,
		       to_char(entry_date, 'YYYY-MM-DD'), summary,
		       COALESCE(source_type, ''), COALESCE(source_id, ''), status,
		       COALESCE(created_by, ''), COALESCE(posted_by, ''),
		       COALESCE(to_char(posted_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
		       to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM journal_entries WHERE id = $1 AND entity_id = $2
	`, journalID, entityID)

	entry, err := scanJournalEntry(row)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	entry.Lines, err = s.listLines(ctx, entityID, journalID)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (s *JournalStore) List(ctx context.Context, query domain.JournalListQuery) ([]domain.JournalEntry, error) {
	baseQuery := ` FROM journal_entries WHERE entity_id = $1`
	args := []any{query.EntityID}
	argIdx := 2

	if query.BookID != "" {
		baseQuery += fmt.Sprintf(` AND book_id = $%d`, argIdx)
		args = append(args, query.BookID)
		argIdx++
	}
	if query.Status != "" {
		baseQuery += fmt.Sprintf(` AND status = $%d`, argIdx)
		args = append(args, query.Status)
		argIdx++
	}
	if query.Period != "" {
		baseQuery += fmt.Sprintf(` AND period = $%d`, argIdx)
		args = append(args, query.Period)
		argIdx++
	}

	selectQuery := `SELECT id, entity_id, book_id, period, COALESCE(voucher_no, ''), voucher_word,
		to_char(entry_date, 'YYYY-MM-DD'), summary,
		COALESCE(source_type, ''), COALESCE(source_id, ''), status,
		COALESCE(created_by, ''), COALESCE(posted_by, ''),
		COALESCE(to_char(posted_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
		to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')` + baseQuery

	selectQuery += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
	args = append(args, query.Limit, query.Offset)

	rows, err := s.db.Pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []domain.JournalEntry
	for rows.Next() {
		entry, err := scanJournalEntry(rows)
		if err != nil {
			return nil, err
		}
		if entry != nil {
			entry.Lines, _ = s.listLines(ctx, query.EntityID, entry.ID)
			entries = append(entries, *entry)
		}
	}
	return entries, rows.Err()
}

func (s *JournalStore) Post(ctx context.Context, entityID, journalID string) (*domain.JournalEntry, error) {
	return s.PostTx(ctx, nil, entityID, journalID)
}

func (s *JournalStore) PostTx(ctx context.Context, tx pgx.Tx, entityID, journalID string) (*domain.JournalEntry, error) {
	exec := s.db.Pool.Exec
	if tx != nil {
		exec = tx.Exec
	}

	tag, err := exec(ctx, `
		UPDATE journal_entries SET status = 'posted', posted_at = now(), updated_at = now()
		WHERE id = $1 AND entity_id = $2 AND status = 'draft'
	`, journalID, entityID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("journal entry not found or not in draft status: %s", journalID)
	}
	return s.Get(ctx, entityID, journalID)
}

func (s *JournalStore) Void(ctx context.Context, entityID, journalID string) (*domain.JournalEntry, error) {
	return s.VoidTx(ctx, nil, entityID, journalID)
}

func (s *JournalStore) VoidTx(ctx context.Context, tx pgx.Tx, entityID, journalID string) (*domain.JournalEntry, error) {
	exec := s.db.Pool.Exec
	if tx != nil {
		exec = tx.Exec
	}
	tag, err := exec(ctx, `
		UPDATE journal_entries SET status = 'voided', updated_at = now()
		WHERE id = $1 AND entity_id = $2 AND status IN ('draft', 'posted')
	`, journalID, entityID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("journal entry not found: %s", journalID)
	}
	return s.Get(ctx, entityID, journalID)
}

func (s *JournalStore) ListPostedLines(ctx context.Context, entityID, bookID, period string) ([]domain.JournalLine, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT jl.id, jl.entity_id, jl.journal_entry_id, jl.account_id, jl.account_code, jl.account_name,
		       jl.direction, jl.debit_amount, jl.credit_amount, jl.currency, jl.line_no, jl.auxiliary
		FROM journal_lines jl
		INNER JOIN journal_entries je ON jl.journal_entry_id = je.id
		WHERE jl.entity_id = $1 AND je.book_id = $2 AND je.period = $3 AND je.status = 'posted'
		ORDER BY jl.account_code
	`, entityID, bookID, period)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []domain.JournalLine
	for rows.Next() {
		var l domain.JournalLine
		if err := rows.Scan(&l.ID, &l.EntityID, &l.JournalEntryID, &l.AccountID, &l.AccountCode, &l.AccountName,
			&l.Direction, &l.DebitAmount, &l.CreditAmount, &l.Currency, &l.LineNo, &l.Auxiliary); err != nil {
			return nil, err
		}
		lines = append(lines, l)
	}
	return lines, rows.Err()
}

func (s *JournalStore) NextVoucherNo(ctx context.Context, entityID, bookID, period string) (string, error) {
	var count int
	err := s.db.Pool.QueryRow(ctx, `
		SELECT count(*) + 1 FROM journal_entries WHERE entity_id = $1 AND book_id = $2 AND period = $3
	`, entityID, bookID, period).Scan(&count)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("记-%s-%03d", period, count), nil
}

func (s *JournalStore) listLines(ctx context.Context, entityID, journalID string) ([]domain.JournalLine, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, entity_id, journal_entry_id, account_id, account_code, account_name,
		       direction, debit_amount, credit_amount, currency, line_no, auxiliary
		FROM journal_lines WHERE entity_id = $1 AND journal_entry_id = $2 ORDER BY line_no
	`, entityID, journalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []domain.JournalLine
	for rows.Next() {
		var l domain.JournalLine
		if err := rows.Scan(&l.ID, &l.EntityID, &l.JournalEntryID, &l.AccountID, &l.AccountCode, &l.AccountName,
			&l.Direction, &l.DebitAmount, &l.CreditAmount, &l.Currency, &l.LineNo, &l.Auxiliary); err != nil {
			return nil, err
		}
		lines = append(lines, l)
	}
	return lines, rows.Err()
}

func scanJournalEntry(row pgx.Row) (*domain.JournalEntry, error) {
	var e domain.JournalEntry
	err := row.Scan(&e.ID, &e.EntityID, &e.BookID, &e.Period, &e.VoucherNo, &e.VoucherWord,
		&e.EntryDate, &e.Summary, &e.SourceType, &e.SourceID, &e.Status,
		&e.CreatedBy, &e.PostedBy, &e.PostedAt, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}
