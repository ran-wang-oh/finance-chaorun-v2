package service

import (
	"context"
	"errors"
	"testing"

	"finance.chao.run/v2/internal/domain"
	"finance.chao.run/v2/internal/provider"

	"github.com/jackc/pgx/v5"
)

// ---- Mock stores ----

type mockBookStore struct {
	getByID    func(ctx context.Context, entityID, bookID string) (*domain.AccountingBook, error)
	getDefault func(ctx context.Context, entityID string) (*domain.AccountingBook, error)
	create     func(ctx context.Context, book *domain.AccountingBook) error
	list       func(ctx context.Context, entityID string) ([]domain.AccountingBook, error)
	lastEntityID string
}

func (m *mockBookStore) Create(ctx context.Context, book *domain.AccountingBook) error {
	m.lastEntityID = book.EntityID
	if m.create != nil {
		return m.create(ctx, book)
	}
	return nil
}
func (m *mockBookStore) GetByID(ctx context.Context, entityID, bookID string) (*domain.AccountingBook, error) {
	m.lastEntityID = entityID
	if m.getByID != nil {
		return m.getByID(ctx, entityID, bookID)
	}
	return nil, nil
}
func (m *mockBookStore) GetDefault(ctx context.Context, entityID string) (*domain.AccountingBook, error) {
	m.lastEntityID = entityID
	if m.getDefault != nil {
		return m.getDefault(ctx, entityID)
	}
	return nil, nil
}
func (m *mockBookStore) List(ctx context.Context, entityID string) ([]domain.AccountingBook, error) {
	m.lastEntityID = entityID
	if m.list != nil {
		return m.list(ctx, entityID)
	}
	return nil, nil
}
func (m *mockBookStore) Update(ctx context.Context, book *domain.AccountingBook) error { return nil }
func (m *mockBookStore) SetDefault(ctx context.Context, entityID, bookID string) error  { return nil }

type mockPeriodStore struct {
	isClosed     func(ctx context.Context, entityID, bookID, period string) (bool, error)
	updateStatus func(ctx context.Context, entityID, bookID, period, from, to string, closedBy *string) (*domain.AccountingPeriod, error)
}

func (m *mockPeriodStore) GetOrCreate(ctx context.Context, p *domain.AccountingPeriod) (*domain.AccountingPeriod, error) {
	return p, nil
}
func (m *mockPeriodStore) Get(ctx context.Context, entityID, bookID, period string) (*domain.AccountingPeriod, error) {
	return nil, nil
}
func (m *mockPeriodStore) ListByBook(ctx context.Context, entityID, bookID string) ([]domain.AccountingPeriod, error) {
	return nil, nil
}
func (m *mockPeriodStore) IsClosed(ctx context.Context, entityID, bookID, period string) (bool, error) {
	if m.isClosed != nil {
		return m.isClosed(ctx, entityID, bookID, period)
	}
	return false, nil
}
func (m *mockPeriodStore) UpdateStatus(ctx context.Context, entityID, bookID, period, from, to string, closedBy *string) (*domain.AccountingPeriod, error) {
	if m.updateStatus != nil {
		return m.updateStatus(ctx, entityID, bookID, period, from, to, closedBy)
	}
	return &domain.AccountingPeriod{EntityID: entityID, BookID: bookID, Period: period, Status: to}, nil
}

type mockInvoiceStore struct {
	create              func(ctx context.Context, inv *domain.Invoice) error
	get                 func(ctx context.Context, entityID, invoiceID string) (*domain.Invoice, error)
	findByInvoiceNo     func(ctx context.Context, entityID, bookID, invoiceNo string) (*domain.Invoice, error)
	approveTx           func(ctx context.Context, tx pgx.Tx, entityID, invoiceID string) (*domain.Invoice, error)
	findByOriginalInvID func(ctx context.Context, entityID, origID string) ([]domain.Invoice, error)
	lastEntityID        string
}

func (m *mockInvoiceStore) Create(ctx context.Context, inv *domain.Invoice) error {
	m.lastEntityID = inv.EntityID
	if m.create != nil {
		return m.create(ctx, inv)
	}
	return nil
}
func (m *mockInvoiceStore) Update(ctx context.Context, inv *domain.Invoice) error                { return nil }
func (m *mockInvoiceStore) Get(ctx context.Context, entityID, invoiceID string) (*domain.Invoice, error) {
	m.lastEntityID = entityID
	if m.get != nil {
		return m.get(ctx, entityID, invoiceID)
	}
	return nil, nil
}
func (m *mockInvoiceStore) List(ctx context.Context, entityID, bookID, period, status string, limit, offset int) ([]domain.Invoice, error) {
	return nil, nil
}
func (m *mockInvoiceStore) FindByInvoiceNo(ctx context.Context, entityID, bookID, invoiceNo string) (*domain.Invoice, error) {
	m.lastEntityID = entityID
	if m.findByInvoiceNo != nil {
		return m.findByInvoiceNo(ctx, entityID, bookID, invoiceNo)
	}
	return nil, errors.New("not found")
}
func (m *mockInvoiceStore) FindByDigitalInvoiceNo(ctx context.Context, entityID, bookID, digitalNo string) (*domain.Invoice, error) {
	return nil, errors.New("not found")
}
func (m *mockInvoiceStore) FindByOriginalInvoiceID(ctx context.Context, entityID, origID string) ([]domain.Invoice, error) {
	m.lastEntityID = entityID
	if m.findByOriginalInvID != nil {
		return m.findByOriginalInvID(ctx, entityID, origID)
	}
	return nil, nil
}
func (m *mockInvoiceStore) Approve(ctx context.Context, entityID, invoiceID string) (*domain.Invoice, error) {
	return nil, nil
}
func (m *mockInvoiceStore) ApproveTx(ctx context.Context, tx pgx.Tx, entityID, invoiceID string) (*domain.Invoice, error) {
	if m.approveTx != nil {
		return m.approveTx(ctx, tx, entityID, invoiceID)
	}
	return &domain.Invoice{ID: invoiceID, EntityID: entityID, Status: domain.StatusApproved}, nil
}
func (m *mockInvoiceStore) Reject(ctx context.Context, entityID, invoiceID string) (*domain.Invoice, error) {
	return nil, nil
}
func (m *mockInvoiceStore) MarkPosted(ctx context.Context, entityID, invoiceID string) (*domain.Invoice, error) {
	return nil, nil
}
func (m *mockInvoiceStore) MarkPostedTx(ctx context.Context, tx pgx.Tx, entityID, invoiceID string) (*domain.Invoice, error) {
	return nil, nil
}
func (m *mockInvoiceStore) UpdateVerificationStatus(ctx context.Context, entityID, invoiceID, status string) error {
	return nil
}
func (m *mockInvoiceStore) UpdateUsageStatus(ctx context.Context, entityID, invoiceID, status string) error {
	return nil
}
func (m *mockInvoiceStore) UpdateRedLetterStatus(ctx context.Context, entityID, invoiceID, status string) error {
	return nil
}
func (m *mockInvoiceStore) UpdateDeductionStatus(ctx context.Context, entityID, invoiceID, status string) error {
	return nil
}
func (m *mockInvoiceStore) ListPostedByPeriod(ctx context.Context, entityID, bookID, period string) ([]domain.Invoice, error) {
	return nil, nil
}

type mockJournalStore struct {
	create        func(ctx context.Context, entry *domain.JournalEntry) error
	get           func(ctx context.Context, entityID, journalID string) (*domain.JournalEntry, error)
	list          func(ctx context.Context, query domain.JournalListQuery) ([]domain.JournalEntry, error)
	post          func(ctx context.Context, entityID, journalID string) (*domain.JournalEntry, error)
	void          func(ctx context.Context, entityID, journalID string) (*domain.JournalEntry, error)
	nextVoucherNo func(ctx context.Context, entityID, bookID, period string) (string, error)
	createTx      func(ctx context.Context, tx pgx.Tx, entry *domain.JournalEntry) error
	postTx        func(ctx context.Context, tx pgx.Tx, entityID, journalID string) (*domain.JournalEntry, error)
	voidTx        func(ctx context.Context, tx pgx.Tx, entityID, journalID string) (*domain.JournalEntry, error)
	lastEntityID  string
}

func (m *mockJournalStore) Create(ctx context.Context, entry *domain.JournalEntry) error {
	m.lastEntityID = entry.EntityID
	if m.create != nil {
		return m.create(ctx, entry)
	}
	return nil
}
func (m *mockJournalStore) CreateTx(ctx context.Context, tx pgx.Tx, entry *domain.JournalEntry) error {
	if m.createTx != nil {
		return m.createTx(ctx, tx, entry)
	}
	return nil
}
func (m *mockJournalStore) Get(ctx context.Context, entityID, journalID string) (*domain.JournalEntry, error) {
	m.lastEntityID = entityID
	if m.get != nil {
		return m.get(ctx, entityID, journalID)
	}
	return nil, nil
}
func (m *mockJournalStore) List(ctx context.Context, query domain.JournalListQuery) ([]domain.JournalEntry, error) {
	if m.list != nil {
		return m.list(ctx, query)
	}
	return nil, nil
}
func (m *mockJournalStore) Post(ctx context.Context, entityID, journalID string) (*domain.JournalEntry, error) {
	if m.post != nil {
		return m.post(ctx, entityID, journalID)
	}
	return nil, nil
}
func (m *mockJournalStore) PostTx(ctx context.Context, tx pgx.Tx, entityID, journalID string) (*domain.JournalEntry, error) {
	if m.postTx != nil {
		return m.postTx(ctx, tx, entityID, journalID)
	}
	return &domain.JournalEntry{ID: journalID, EntityID: entityID, Status: domain.JournalStatusPosted}, nil
}
func (m *mockJournalStore) Void(ctx context.Context, entityID, journalID string) (*domain.JournalEntry, error) {
	if m.void != nil {
		return m.void(ctx, entityID, journalID)
	}
	return &domain.JournalEntry{ID: journalID, EntityID: entityID, Status: domain.JournalStatusVoided}, nil
}
func (m *mockJournalStore) VoidTx(ctx context.Context, tx pgx.Tx, entityID, journalID string) (*domain.JournalEntry, error) {
	if m.voidTx != nil {
		return m.voidTx(ctx, tx, entityID, journalID)
	}
	return &domain.JournalEntry{ID: journalID, EntityID: entityID, Status: domain.JournalStatusVoided}, nil
}
func (m *mockJournalStore) ListPostedLines(ctx context.Context, entityID, bookID, period string) ([]domain.JournalLine, error) {
	return nil, nil
}
func (m *mockJournalStore) NextVoucherNo(ctx context.Context, entityID, bookID, period string) (string, error) {
	if m.nextVoucherNo != nil {
		return m.nextVoucherNo(ctx, entityID, bookID, period)
	}
	return "1", nil
}

type mockAuditStore struct {
	entries []domain.AuditEntry
}

func (m *mockAuditStore) Append(ctx context.Context, e *domain.AuditEntry) error {
	m.entries = append(m.entries, *e)
	return nil
}
func (m *mockAuditStore) List(ctx context.Context, query domain.AuditListQuery) ([]domain.AuditEntry, error) {
	return m.entries, nil
}
func (m *mockAuditStore) lastEntry() *domain.AuditEntry {
	if len(m.entries) == 0 {
		return nil
	}
	return &m.entries[len(m.entries)-1]
}

type mockAccountStore struct {
	create    func(ctx context.Context, a *domain.ChartAccount) error
	getByCode func(ctx context.Context, entityID, bookID, code string) (*domain.ChartAccount, error)
}

func (m *mockAccountStore) Create(ctx context.Context, a *domain.ChartAccount) error {
	if m.create != nil {
		return m.create(ctx, a)
	}
	return nil
}
func (m *mockAccountStore) GetByCode(ctx context.Context, entityID, bookID, code string) (*domain.ChartAccount, error) {
	if m.getByCode != nil {
		return m.getByCode(ctx, entityID, bookID, code)
	}
	return &domain.ChartAccount{Code: code, Name: code, BalanceType: "debit"}, nil
}
func (m *mockAccountStore) List(ctx context.Context, query domain.AccountListQuery) ([]domain.ChartAccount, int, error) {
	return nil, 0, nil
}
func (m *mockAccountStore) Update(ctx context.Context, a *domain.ChartAccount) error { return nil }
func (m *mockAccountStore) Delete(ctx context.Context, entityID, accountID string) error {
	return nil
}

// ---- Test helpers ----

func ptrStr(s string) *string { return &s }

func ctxWithTrace() context.Context {
	return provider.ContextWithTraceID(context.Background(), "test-trace-001")
}

// ---- 1. Book ownership tests ----

func TestResolveBook_ExplicitBookOwnership(t *testing.T) {
	bookStore := &mockBookStore{
		getByID: func(ctx context.Context, entityID, bookID string) (*domain.AccountingBook, error) {
			if entityID == "e1" && bookID == "book-1" {
				return &domain.AccountingBook{ID: "book-1", EntityID: "e1"}, nil
			}
			return nil, errors.New("not found")
		},
	}
	svc := New(nil).WithBooks(bookStore)

	t.Run("book belongs to entity", func(t *testing.T) {
		id, err := svc.ResolveBook(ctxWithTrace(), "e1", "book-1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if id != "book-1" {
			t.Errorf("expected book-1, got %s", id)
		}
	})

	t.Run("book does not belong to entity", func(t *testing.T) {
		_, err := svc.ResolveBook(ctxWithTrace(), "e1", "book-other")
		if err == nil {
			t.Fatal("expected error for cross-entity book access")
		}
		if !errors.Is(err, ErrBookScopeMismatch) {
			t.Errorf("expected ErrBookScopeMismatch, got %v", err)
		}
	})

	t.Run("bookID empty falls back to default", func(t *testing.T) {
		bookStore.getDefault = func(ctx context.Context, entityID string) (*domain.AccountingBook, error) {
			return &domain.AccountingBook{ID: "default-book", EntityID: entityID}, nil
		}
		id, err := svc.ResolveBook(ctxWithTrace(), "e1", "")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if id != "default-book" {
			t.Errorf("expected default-book, got %s", id)
		}
	})

	t.Run("entityID passed to GetByID", func(t *testing.T) {
		bookStore.getByID = func(ctx context.Context, entityID, bookID string) (*domain.AccountingBook, error) {
			if entityID != "e1" {
				t.Errorf("expected entityID e1, got %s", entityID)
			}
			return &domain.AccountingBook{ID: bookID, EntityID: entityID}, nil
		}
		_, _ = svc.ResolveBook(ctxWithTrace(), "e1", "book-1")
	})
}

func TestListBooks_EntityScoped(t *testing.T) {
	bookStore := &mockBookStore{
		list: func(ctx context.Context, entityID string) ([]domain.AccountingBook, error) {
			if entityID != "e1" {
				return nil, errors.New("wrong entity")
			}
			return []domain.AccountingBook{
				{ID: "b1", EntityID: "e1", Name: "Book 1"},
				{ID: "b2", EntityID: "e1", Name: "Book 2"},
			}, nil
		},
	}
	svc := New(nil).WithBooks(bookStore)

	books, err := svc.ListBooks(ctxWithTrace(), "e1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(books) != 2 {
		t.Errorf("expected 2 books, got %d", len(books))
	}
	if books[0].EntityID != "e1" {
		t.Errorf("expected entityID e1, got %s", books[0].EntityID)
	}
}

func TestGetBook_EntityScoped(t *testing.T) {
	var capturedEntityID string
	bookStore := &mockBookStore{
		getByID: func(ctx context.Context, entityID, bookID string) (*domain.AccountingBook, error) {
			capturedEntityID = entityID
			return &domain.AccountingBook{ID: bookID, EntityID: entityID}, nil
		},
	}
	svc := New(nil).WithBooks(bookStore)

	_, _ = svc.GetBook(ctxWithTrace(), "e1", "book-1")
	if capturedEntityID != "e1" {
		t.Errorf("expected entityID e1, got %s", capturedEntityID)
	}
}

// ---- 2. Period closed/locked write denial tests ----

func TestCreateInvoiceDraft_PeriodClosed(t *testing.T) {
	periods := &mockPeriodStore{
		isClosed: func(ctx context.Context, entityID, bookID, period string) (bool, error) {
			return true, nil
		},
	}
	invoices := &mockInvoiceStore{
		findByInvoiceNo: func(ctx context.Context, entityID, bookID, invoiceNo string) (*domain.Invoice, error) {
			return nil, errors.New("not found")
		},
	}
	svc := New(nil).WithPeriods(periods).WithInvoices(invoices)

	inv := &domain.Invoice{
		InvoiceNo:        "INV-001",
		Direction:        domain.DirectionInput,
		IssueDate:        "2026-03-15",
		SellerName:       "Test Corp",
		AmountWithoutTax: 100,
		TaxAmount:        13,
		AmountWithTax:    113,
	}
	_, err := svc.CreateInvoiceDraft(ctxWithTrace(), "e1", "book-1", "actor-1", inv)
	if !errors.Is(err, ErrPeriodClosed) {
		t.Fatalf("expected ErrPeriodClosed, got %v", err)
	}
}

func TestCreateInvoiceDraft_PeriodOpen(t *testing.T) {
	periods := &mockPeriodStore{
		isClosed: func(ctx context.Context, entityID, bookID, period string) (bool, error) {
			return false, nil
		},
	}
	invoices := &mockInvoiceStore{
		findByInvoiceNo: func(ctx context.Context, entityID, bookID, invoiceNo string) (*domain.Invoice, error) {
			return nil, errors.New("not found")
		},
	}
	audit := &mockAuditStore{}
	svc := New(nil).WithPeriods(periods).WithInvoices(invoices).WithAuditLog(audit)

	inv := &domain.Invoice{
		InvoiceNo:        "INV-002",
		Direction:        domain.DirectionInput,
		IssueDate:        "2026-03-15",
		SellerName:       "Test Corp",
		AmountWithoutTax: 100,
		TaxAmount:        13,
		AmountWithTax:    113,
	}
	result, err := svc.CreateInvoiceDraft(ctxWithTrace(), "e1", "book-1", "actor-1", inv)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Status != domain.StatusPendingReview {
		t.Errorf("expected pending_review status, got %s", result.Status)
	}
	if result.EntityID != "e1" {
		t.Errorf("expected entityID e1, got %s", result.EntityID)
	}
}

func TestCreateJournalDraft_PeriodClosed(t *testing.T) {
	periods := &mockPeriodStore{
		isClosed: func(ctx context.Context, entityID, bookID, period string) (bool, error) {
			return true, nil
		},
	}
	journals := &mockJournalStore{
		nextVoucherNo: func(ctx context.Context, entityID, bookID, period string) (string, error) {
			return "5", nil
		},
	}
	svc := New(nil).WithPeriods(periods).WithJournals(journals)

	entry := &domain.JournalEntry{
		BookID: "book-1",
		Period: "2026-03",
		Lines: []domain.JournalLine{
			{AccountCode: "1001", Direction: domain.DirectionDebit, DebitAmount: 100, CreditAmount: 0},
			{AccountCode: "6001", Direction: domain.DirectionCredit, DebitAmount: 0, CreditAmount: 100},
		},
	}
	_, err := svc.CreateJournalDraft(ctxWithTrace(), "e1", "book-1", "2026-03", "actor-1", entry)
	if !errors.Is(err, ErrPeriodClosed) {
		t.Fatalf("expected ErrPeriodClosed, got %v", err)
	}
}

func TestApproveInvoice_PeriodClosed(t *testing.T) {
	periods := &mockPeriodStore{
		isClosed: func(ctx context.Context, entityID, bookID, period string) (bool, error) {
			return true, nil
		},
	}
	invoices := &mockInvoiceStore{
		get: func(ctx context.Context, entityID, invoiceID string) (*domain.Invoice, error) {
			return &domain.Invoice{
				ID: invoiceID, EntityID: entityID, BookID: "book-1",
				IssueDate: "2026-03-15", Status: domain.StatusPendingReview,
				Direction: domain.DirectionInput, AmountWithoutTax: 100, TaxAmount: 13, AmountWithTax: 113,
			}, nil
		},
	}
	svc := New(nil).WithPeriods(periods).WithInvoices(invoices)

	_, _, err := svc.ApproveInvoice(ctxWithTrace(), "e1", "inv-1", "actor-1")
	if !errors.Is(err, ErrPeriodClosed) {
		t.Fatalf("expected ErrPeriodClosed, got %v", err)
	}
}

func TestPostJournalEntry_PeriodClosed(t *testing.T) {
	periods := &mockPeriodStore{
		isClosed: func(ctx context.Context, entityID, bookID, period string) (bool, error) {
			return true, nil
		},
	}
	journals := &mockJournalStore{
		get: func(ctx context.Context, entityID, journalID string) (*domain.JournalEntry, error) {
			return &domain.JournalEntry{
				ID: journalID, EntityID: entityID, BookID: "book-1",
				Period: "2026-03", Status: domain.JournalStatusDraft,
				Lines: []domain.JournalLine{
					{AccountCode: "1001", Direction: domain.DirectionDebit, DebitAmount: 100},
					{AccountCode: "6001", Direction: domain.DirectionCredit, CreditAmount: 100},
				},
			}, nil
		},
	}
	svc := New(nil).WithPeriods(periods).WithJournals(journals)

	_, err := svc.PostJournalEntry(ctxWithTrace(), "e1", "je-1", "actor-1")
	if !errors.Is(err, ErrPeriodClosed) {
		t.Fatalf("expected ErrPeriodClosed, got %v", err)
	}
}

func TestVoidJournalEntry_PeriodClosed(t *testing.T) {
	periods := &mockPeriodStore{
		isClosed: func(ctx context.Context, entityID, bookID, period string) (bool, error) {
			return true, nil
		},
	}
	journals := &mockJournalStore{
		get: func(ctx context.Context, entityID, journalID string) (*domain.JournalEntry, error) {
			return &domain.JournalEntry{
				ID: journalID, EntityID: entityID, BookID: "book-1",
				Period: "2026-03", Status: domain.JournalStatusDraft,
			}, nil
		},
	}
	svc := New(nil).WithPeriods(periods).WithJournals(journals)

	_, err := svc.VoidJournalEntry(ctxWithTrace(), "e1", "je-1", "actor-1")
	if !errors.Is(err, ErrPeriodClosed) {
		t.Fatalf("expected ErrPeriodClosed, got %v", err)
	}
}

// ---- 3. Journal debit/credit balance tests ----

func TestValidateBalanced(t *testing.T) {
	t.Run("balanced", func(t *testing.T) {
		entry := &domain.JournalEntry{
			Lines: []domain.JournalLine{
				{DebitAmount: 100, CreditAmount: 0},
				{DebitAmount: 0, CreditAmount: 100},
			},
		}
		if err := validateBalanced(entry); err != nil {
			t.Errorf("expected balanced, got error: %v", err)
		}
	})

	t.Run("balanced with cents", func(t *testing.T) {
		entry := &domain.JournalEntry{
			Lines: []domain.JournalLine{
				{DebitAmount: 100.50, CreditAmount: 0},
				{DebitAmount: 0, CreditAmount: 50.25},
				{DebitAmount: 0, CreditAmount: 50.25},
			},
		}
		if err := validateBalanced(entry); err != nil {
			t.Errorf("expected balanced, got error: %v", err)
		}
	})

	t.Run("unbalanced", func(t *testing.T) {
		entry := &domain.JournalEntry{
			Lines: []domain.JournalLine{
				{DebitAmount: 100, CreditAmount: 0},
				{DebitAmount: 0, CreditAmount: 50},
			},
		}
		err := validateBalanced(entry)
		if err == nil {
			t.Fatal("expected error for unbalanced entry")
		}
		if !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("expected ErrInvalidRequest, got %v", err)
		}
	})

	t.Run("zero sum", func(t *testing.T) {
		entry := &domain.JournalEntry{
			Lines: []domain.JournalLine{
				{DebitAmount: 0, CreditAmount: 0},
			},
		}
		if err := validateBalanced(entry); err != nil {
			t.Errorf("expected balanced for zero lines, got error: %v", err)
		}
	})
}

func TestCreateJournalDraft_Unbalanced(t *testing.T) {
	journals := &mockJournalStore{
		nextVoucherNo: func(ctx context.Context, entityID, bookID, period string) (string, error) {
			return "1", nil
		},
	}
	svc := New(nil).WithJournals(journals)

	entry := &domain.JournalEntry{
		BookID: "book-1",
		Period: "2026-03",
		Lines: []domain.JournalLine{
			{AccountCode: "1001", Direction: domain.DirectionDebit, DebitAmount: 100, CreditAmount: 0},
		},
	}
	_, err := svc.CreateJournalDraft(ctxWithTrace(), "e1", "book-1", "2026-03", "actor-1", entry)
	if err == nil {
		t.Fatal("expected error for unbalanced journal entry")
	}
}

func TestCreateJournalDraft_NoLines(t *testing.T) {
	svc := New(nil)
	entry := &domain.JournalEntry{
		BookID: "book-1",
		Period: "2026-03",
		Lines:  []domain.JournalLine{},
	}
	_, err := svc.CreateJournalDraft(ctxWithTrace(), "e1", "book-1", "2026-03", "actor-1", entry)
	if err == nil {
		t.Fatal("expected error for journal entry with no lines")
	}
}

// ---- 4. Duplicate invoice tests ----

func TestCreateInvoiceDraft_DuplicateInvoiceNo(t *testing.T) {
	existing := &domain.Invoice{
		ID: "inv-existing", EntityID: "e1", BookID: "book-1",
		InvoiceNo: "INV-DUP", Status: domain.StatusPendingReview,
	}
	invoices := &mockInvoiceStore{
		findByInvoiceNo: func(ctx context.Context, entityID, bookID, invoiceNo string) (*domain.Invoice, error) {
			return existing, nil
		},
	}
	svc := New(nil).WithInvoices(invoices)

	inv := &domain.Invoice{
		InvoiceNo:        "INV-DUP",
		Direction:        domain.DirectionInput,
		IssueDate:        "2026-03-15",
		SellerName:       "Test Corp",
		AmountWithoutTax: 100,
		TaxAmount:        13,
		AmountWithTax:    113,
	}
	result, err := svc.CreateInvoiceDraft(ctxWithTrace(), "e1", "book-1", "actor-1", inv)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.ID != existing.ID {
		t.Errorf("expected existing invoice %s, got %s", existing.ID, result.ID)
	}
}

func TestCreateInvoiceDraft_InvalidAmounts(t *testing.T) {
	invoices := &mockInvoiceStore{
		findByInvoiceNo: func(ctx context.Context, entityID, bookID, invoiceNo string) (*domain.Invoice, error) {
			return nil, errors.New("not found")
		},
	}
	svc := New(nil).WithInvoices(invoices)

	t.Run("amount mismatch", func(t *testing.T) {
		inv := &domain.Invoice{
			InvoiceNo:        "INV-003",
			Direction:        domain.DirectionInput,
			IssueDate:        "2026-03-15",
			SellerName:       "Test Corp",
			AmountWithoutTax: 100,
			TaxAmount:        20,
			AmountWithTax:    113,
		}
		_, err := svc.CreateInvoiceDraft(ctxWithTrace(), "e1", "book-1", "actor-1", inv)
		if err == nil {
			t.Fatal("expected error for amount mismatch")
		}
	})

	t.Run("negative amount accepted in draft", func(t *testing.T) {
		inv := &domain.Invoice{
			InvoiceNo:        "INV-004",
			Direction:        domain.DirectionInput,
			IssueDate:        "2026-03-15",
			SellerName:       "Test Corp",
			AmountWithoutTax: -100,
			TaxAmount:        -13,
			AmountWithTax:    -113,
		}
		// CreateInvoiceDraft doesn't reject negative amounts —
		// that check is in buildJournalDraft during approval.
		result, err := svc.CreateInvoiceDraft(ctxWithTrace(), "e1", "book-1", "actor-1", inv)
		if err != nil {
			t.Fatalf("CreateInvoiceDraft should accept negative amounts (validated at approve time): %v", err)
		}
		if result.Status != domain.StatusPendingReview {
			t.Errorf("expected pending_review, got %s", result.Status)
		}
	})

	t.Run("missing invoice_no", func(t *testing.T) {
		inv := &domain.Invoice{
			Direction:        domain.DirectionInput,
			IssueDate:        "2026-03-15",
			AmountWithoutTax: 100,
			TaxAmount:        13,
			AmountWithTax:    113,
		}
		_, err := svc.CreateInvoiceDraft(ctxWithTrace(), "e1", "book-1", "actor-1", inv)
		if err == nil {
			t.Fatal("expected error for missing invoice_no")
		}
	})
}

// ---- 5. Entity-scoped query tests ----

func TestGetInvoice_EntityScoped(t *testing.T) {
	var capturedEntityID string
	invoices := &mockInvoiceStore{
		get: func(ctx context.Context, entityID, invoiceID string) (*domain.Invoice, error) {
			capturedEntityID = entityID
			return &domain.Invoice{ID: invoiceID, EntityID: entityID}, nil
		},
	}
	svc := New(nil).WithInvoices(invoices)

	_, _ = svc.GetInvoice(ctxWithTrace(), "e1", "inv-1")
	if capturedEntityID != "e1" {
		t.Errorf("expected entityID e1 passed to store, got %s", capturedEntityID)
	}
}

func TestGetJournalEntry_EntityScoped(t *testing.T) {
	var capturedEntityID string
	journals := &mockJournalStore{
		get: func(ctx context.Context, entityID, journalID string) (*domain.JournalEntry, error) {
			capturedEntityID = entityID
			return &domain.JournalEntry{ID: journalID, EntityID: entityID}, nil
		},
	}
	svc := New(nil).WithJournals(journals)

	_, _ = svc.GetJournalEntry(ctxWithTrace(), "e1", "je-1")
	if capturedEntityID != "e1" {
		t.Errorf("expected entityID e1 passed to store, got %s", capturedEntityID)
	}
}

func TestCreateInvoiceDraft_EntityScoped(t *testing.T) {
	var capturedEntityID string
	invoices := &mockInvoiceStore{
		findByInvoiceNo: func(ctx context.Context, entityID, bookID, invoiceNo string) (*domain.Invoice, error) {
			capturedEntityID = entityID
			return nil, errors.New("not found")
		},
		create: func(ctx context.Context, inv *domain.Invoice) error {
			if inv.EntityID != capturedEntityID {
				t.Errorf("invoice EntityID %s doesn't match captured %s", inv.EntityID, capturedEntityID)
			}
			return nil
		},
	}
	svc := New(nil).WithInvoices(invoices)

	_, _ = svc.CreateInvoiceDraft(ctxWithTrace(), "e2", "book-2", "actor-1", &domain.Invoice{
		InvoiceNo: "INV-E2", Direction: domain.DirectionInput, IssueDate: "2026-03-15",
		SellerName: "E2 Corp", AmountWithoutTax: 100, TaxAmount: 13, AmountWithTax: 113,
	})
	if capturedEntityID != "e2" {
		t.Errorf("expected entityID e2 passed to store, got %s", capturedEntityID)
	}
}

func TestFindByInvoiceNo_EntityScoped(t *testing.T) {
	var capturedEntityID string
	invoices := &mockInvoiceStore{
		findByInvoiceNo: func(ctx context.Context, entityID, bookID, invoiceNo string) (*domain.Invoice, error) {
			capturedEntityID = entityID
			return nil, errors.New("not found")
		},
	}
	svc := New(nil).WithInvoices(invoices)

	_, _ = svc.CreateInvoiceDraft(ctxWithTrace(), "e1", "book-1", "actor-1", &domain.Invoice{
		InvoiceNo: "INV-SCOPED", Direction: domain.DirectionInput, IssueDate: "2026-03-15",
		SellerName: "Corp", AmountWithoutTax: 100, TaxAmount: 13, AmountWithTax: 113,
	})
	if capturedEntityID != "e1" {
		t.Errorf("expected entityID e1 in FindByInvoiceNo, got %s", capturedEntityID)
	}
}

// ---- 6. Domain audit write tests ----

func TestLogAudit_CreateBook(t *testing.T) {
	audit := &mockAuditStore{}
	books := &mockBookStore{}
	svc := New(nil).WithBooks(books).WithAuditLog(audit)

	ctx := ctxWithTrace()
	ctx = provider.ContextWithCapabilityID(ctx, "finance.book.create")
	ctx = provider.ContextWithIdempotencyKey(ctx, "idem-book-1")
	ctx = provider.ContextWithActorID(ctx, "actor-1")

	err := svc.CreateBook(ctx, &domain.AccountingBook{
		EntityID: "e1", Name: "Test Book", BaseCurrency: "CNY",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	last := audit.lastEntry()
	if last == nil {
		t.Fatal("expected audit entry to be written")
	}
	if last.Action != domain.AuditActionBookCreate {
		t.Errorf("expected action %s, got %s", domain.AuditActionBookCreate, last.Action)
	}
	if last.EntityID != "e1" {
		t.Errorf("expected entityID e1, got %s", last.EntityID)
	}
	if last.ObjectType != "book" {
		t.Errorf("expected objectType book, got %s", last.ObjectType)
	}
	if last.TraceID != "test-trace-001" {
		t.Errorf("expected traceID test-trace-001, got %s", last.TraceID)
	}
	if last.CapabilityID != "finance.book.create" {
		t.Errorf("expected capabilityID, got %s", last.CapabilityID)
	}
}

func TestLogAudit_CreateInvoiceDraft(t *testing.T) {
	audit := &mockAuditStore{}
	invoices := &mockInvoiceStore{
		findByInvoiceNo: func(ctx context.Context, entityID, bookID, invoiceNo string) (*domain.Invoice, error) {
			return nil, errors.New("not found")
		},
	}
	svc := New(nil).WithInvoices(invoices).WithAuditLog(audit)

	ctx := ctxWithTrace()
	ctx = provider.ContextWithCapabilityID(ctx, "finance.invoice.create_draft")
	ctx = provider.ContextWithIdempotencyKey(ctx, "idem-inv-1")

	_, err := svc.CreateInvoiceDraft(ctx, "e1", "book-1", "actor-1", &domain.Invoice{
		InvoiceNo: "INV-AUDIT", Direction: domain.DirectionInput, IssueDate: "2026-03-15",
		SellerName: "Audit Corp", AmountWithoutTax: 100, TaxAmount: 13, AmountWithTax: 113,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	last := audit.lastEntry()
	if last == nil {
		t.Fatal("expected audit entry to be written")
	}
	if last.Action != domain.AuditActionInvoiceCreateDraft {
		t.Errorf("expected action %s, got %s", domain.AuditActionInvoiceCreateDraft, last.Action)
	}
	if last.TraceID != "test-trace-001" {
		t.Errorf("expected traceID, got %s", last.TraceID)
	}
}

func TestLogAudit_CreateJournalDraft(t *testing.T) {
	audit := &mockAuditStore{}
	journals := &mockJournalStore{
		nextVoucherNo: func(ctx context.Context, entityID, bookID, period string) (string, error) {
			return "10", nil
		},
	}
	svc := New(nil).WithJournals(journals).WithAuditLog(audit)

	ctx := ctxWithTrace()
	ctx = provider.ContextWithCapabilityID(ctx, "finance.journal.create_draft")
	ctx = provider.ContextWithIdempotencyKey(ctx, "idem-je-1")

	_, err := svc.CreateJournalDraft(ctx, "e1", "book-1", "2026-03", "actor-1", &domain.JournalEntry{
		Summary: "test entry",
		Lines: []domain.JournalLine{
			{AccountCode: "1001", Direction: domain.DirectionDebit, DebitAmount: 50},
			{AccountCode: "6001", Direction: domain.DirectionCredit, CreditAmount: 50},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	last := audit.lastEntry()
	if last == nil {
		t.Fatal("expected audit entry to be written")
	}
	if last.Action != domain.AuditActionJournalCreateDraft {
		t.Errorf("expected action %s, got %s", domain.AuditActionJournalCreateDraft, last.Action)
	}
	if last.EntityID != "e1" {
		t.Errorf("expected entityID e1, got %s", last.EntityID)
	}
}

func TestLogAudit_CloseAndLockPeriod(t *testing.T) {
	audit := &mockAuditStore{}
	periods := &mockPeriodStore{
		updateStatus: func(ctx context.Context, entityID, bookID, period, from, to string, closedBy *string) (*domain.AccountingPeriod, error) {
			return &domain.AccountingPeriod{ID: "p1", EntityID: entityID, BookID: bookID, Period: period, Status: to}, nil
		},
	}
	svc := New(nil).WithPeriods(periods).WithAuditLog(audit)

	ctx := ctxWithTrace()
	ctx = provider.ContextWithCapabilityID(ctx, "finance.period.lock")

	// ClosePeriod requires book_id; the period update status transitions open->closing->closed
	// ClosePeriod does a close check which needs invoices and journals populated
	// We test LockPeriod directly since it's simpler
	_, err := svc.LockPeriod(ctx, "e1", "book-1", "2026-03", "actor-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	last := audit.lastEntry()
	if last == nil {
		t.Fatal("expected audit entry to be written")
	}
	if last.Action != domain.AuditActionPeriodLock {
		t.Errorf("expected action %s, got %s", domain.AuditActionPeriodLock, last.Action)
	}
	if last.ActorID != "actor-1" {
		t.Errorf("expected actorID actor-1, got %s", last.ActorID)
	}
}

func TestLogAudit_ActorFromContext(t *testing.T) {
	audit := &mockAuditStore{}
	books := &mockBookStore{}
	svc := New(nil).WithBooks(books).WithAuditLog(audit)

	ctx := ctxWithTrace()
	ctx = provider.ContextWithActorID(ctx, "context-actor")

	err := svc.CreateBook(ctx, &domain.AccountingBook{
		EntityID: "e1", Name: "Actor Test", BaseCurrency: "CNY",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	last := audit.lastEntry()
	if last.ActorID != "context-actor" {
		t.Errorf("expected actorID from context, got %s", last.ActorID)
	}
}

func TestLogAudit_MultipleEntries(t *testing.T) {
	audit := &mockAuditStore{}
	books := &mockBookStore{}
	invoices := &mockInvoiceStore{
		findByInvoiceNo: func(ctx context.Context, entityID, bookID, invoiceNo string) (*domain.Invoice, error) {
			return nil, errors.New("not found")
		},
	}
	journals := &mockJournalStore{
		nextVoucherNo: func(ctx context.Context, entityID, bookID, period string) (string, error) {
			return "1", nil
		},
	}
	svc := New(nil).WithBooks(books).WithInvoices(invoices).WithJournals(journals).WithAuditLog(audit)

	ctx := ctxWithTrace()

	// Create book
	svc.CreateBook(ctx, &domain.AccountingBook{EntityID: "e1", Name: "B1", BaseCurrency: "CNY"})
	// Create invoice
	svc.CreateInvoiceDraft(ctx, "e1", "b1", "a1", &domain.Invoice{
		InvoiceNo: "INV-M1", Direction: domain.DirectionInput, IssueDate: "2026-03-15",
		SellerName: "C1", AmountWithoutTax: 100, TaxAmount: 13, AmountWithTax: 113,
	})
	// Create journal
	svc.CreateJournalDraft(ctx, "e1", "b1", "2026-03", "a1", &domain.JournalEntry{
		Summary: "multi-test",
		Lines: []domain.JournalLine{
			{AccountCode: "1001", Direction: domain.DirectionDebit, DebitAmount: 200},
			{AccountCode: "6001", Direction: domain.DirectionCredit, CreditAmount: 200},
		},
	})

	if len(audit.entries) != 3 {
		t.Fatalf("expected 3 audit entries, got %d", len(audit.entries))
	}

	actions := []string{
		domain.AuditActionBookCreate,
		domain.AuditActionInvoiceCreateDraft,
		domain.AuditActionJournalCreateDraft,
	}
	for i, want := range actions {
		if audit.entries[i].Action != want {
			t.Errorf("entry %d: expected action %s, got %s", i, want, audit.entries[i].Action)
		}
	}
}

// ---- 7. Nil store guard tests ----

func TestNilStore_ReturnsErrNotConfigured(t *testing.T) {
	svc := New(nil)

	_, err := svc.ListBooks(ctxWithTrace(), "e1")
	if !errors.Is(err, ErrNotConfigured) {
		t.Errorf("ListBooks: expected ErrNotConfigured, got %v", err)
	}

	_, err = svc.ListPeriods(ctxWithTrace(), "e1", "b1")
	if !errors.Is(err, ErrNotConfigured) {
		t.Errorf("ListPeriods: expected ErrNotConfigured, got %v", err)
	}

	_, err = svc.ListAuditLog(ctxWithTrace(), domain.AuditListQuery{})
	if !errors.Is(err, ErrNotConfigured) {
		t.Errorf("ListAuditLog: expected ErrNotConfigured, got %v", err)
	}
}

func TestGetInvoice_EmptyID(t *testing.T) {
	invoices := &mockInvoiceStore{}
	svc := New(nil).WithInvoices(invoices)

	_, err := svc.GetInvoice(ctxWithTrace(), "e1", "")
	if err == nil {
		t.Fatal("expected error for empty invoice id")
	}
	if !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("expected ErrInvalidRequest, got %v", err)
	}
}

func TestGetJournalEntry_EmptyID(t *testing.T) {
	journals := &mockJournalStore{}
	svc := New(nil).WithJournals(journals)

	_, err := svc.GetJournalEntry(ctxWithTrace(), "e1", "")
	if err == nil {
		t.Fatal("expected error for empty journal id")
	}
}

// ---- 8. roundMoney and moneyEqual helpers ----

func TestRoundMoney(t *testing.T) {
	tests := []struct {
		in, want float64
	}{
		{100.005, 100.01},
		{100.004, 100.00},
		{0, 0},
		{-50.005, -50.01},
		{1.999, 2.00},
	}
	for _, tc := range tests {
		got := roundMoney(tc.in)
		if got != tc.want {
			t.Errorf("roundMoney(%f) = %f, want %f", tc.in, got, tc.want)
		}
	}
}

func TestMoneyEqual(t *testing.T) {
	if !moneyEqual(100.001, 100.002) {
		t.Error("100.001 and 100.002 should be equal within tolerance")
	}
	if moneyEqual(100, 100.01) {
		t.Error("100 and 100.01 should not be equal")
	}
}

// ---- 9. accountingPeriod helper ----

func TestAccountingPeriod(t *testing.T) {
	if got := accountingPeriod("2026-03-15"); got != "2026-03" {
		t.Errorf("expected 2026-03, got %s", got)
	}
	if got := accountingPeriod("bad"); got == "bad" {
		t.Errorf("expected fallback for invalid date")
	}
}

// ---- 10. buildReversal balances correctly ----

func TestBuildReversal_Balanced(t *testing.T) {
	journals := &mockJournalStore{
		nextVoucherNo: func(ctx context.Context, entityID, bookID, period string) (string, error) {
			return "99", nil
		},
	}
	svc := New(nil).WithJournals(journals)

	original := &domain.JournalEntry{
		EntityID: "e1", BookID: "b1", Period: "2026-03",
		Summary: "original entry",
		Lines: []domain.JournalLine{
			{AccountCode: "1001", AccountName: "Cash", Direction: domain.DirectionDebit, DebitAmount: 100, CreditAmount: 0},
			{AccountCode: "6001", AccountName: "Revenue", Direction: domain.DirectionCredit, DebitAmount: 0, CreditAmount: 100},
		},
	}

	rev := svc.buildReversal(original)

	if rev.Summary != "红字冲销: original entry" {
		t.Errorf("unexpected summary: %s", rev.Summary)
	}
	if len(rev.Lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(rev.Lines))
	}

	// Reversal should flip debits and credits
	if rev.Lines[0].DebitAmount != 0 || rev.Lines[0].CreditAmount != 100 {
		t.Errorf("line 0 should have credit 100 (was debit 100), got debit=%f credit=%f", rev.Lines[0].DebitAmount, rev.Lines[0].CreditAmount)
	}
	if rev.Lines[1].DebitAmount != 100 || rev.Lines[1].CreditAmount != 0 {
		t.Errorf("line 1 should have debit 100 (was credit 100), got debit=%f credit=%f", rev.Lines[1].DebitAmount, rev.Lines[1].CreditAmount)
	}

	// Reversal must be balanced
	if err := validateBalanced(rev); err != nil {
		t.Errorf("reversal is not balanced: %v", err)
	}
}
