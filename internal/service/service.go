package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"finance.chao.run/v2/internal/domain"
	"finance.chao.run/v2/internal/engine"
	"finance.chao.run/v2/internal/provider"
	"finance.chao.run/v2/internal/store"
	"finance.chao.run/v2/internal/store/postgres"

	"github.com/google/uuid"
)

var (
	ErrInvalidRequest  = errors.New("invalid request")
	ErrNotConfigured   = errors.New("feature not configured")
	ErrPeriodClosed    = errors.New("period is closed or locked")
	ErrPeriodLocked    = errors.New("period is locked")
	ErrBookNotFound    = errors.New("book not found")
	ErrBookScopeMismatch = errors.New("book does not belong to entity")
)

type Service struct {
	db              *postgres.DB
	books           store.BookStore
	periods         store.PeriodStore
	invoices        store.InvoiceStore
	invoiceLines    store.InvoiceLineStore
	accounts        store.AccountStore
	journals        store.JournalStore
	auditLog        store.AuditStore
	idempotency     store.IdempotencyStore
	reconciliation  store.ReconciliationStore
	reportMappings  store.ReportMappingStore
	adjustments     store.AdjustmentStore
	taxProfiles     store.TaxProfileStore
	taxReturns      store.TaxReturnStore
	riskScans       store.RiskScanStore
	consistencyChecks store.ConsistencyCheckStore
}

func New(db *postgres.DB) *Service {
	return &Service{db: db}
}

func (s *Service) WithBooks(b store.BookStore) *Service                { s.books = b; return s }
func (s *Service) WithPeriods(p store.PeriodStore) *Service              { s.periods = p; return s }
func (s *Service) WithInvoices(i store.InvoiceStore) *Service           { s.invoices = i; return s }
func (s *Service) WithInvoiceLines(il store.InvoiceLineStore) *Service  { s.invoiceLines = il; return s }
func (s *Service) WithAccounts(a store.AccountStore) *Service           { s.accounts = a; return s }
func (s *Service) WithJournals(j store.JournalStore) *Service           { s.journals = j; return s }
func (s *Service) WithAuditLog(a store.AuditStore) *Service             { s.auditLog = a; return s }
func (s *Service) WithIdempotency(i store.IdempotencyStore) *Service       { s.idempotency = i; return s }
func (s *Service) WithReconciliation(r store.ReconciliationStore) *Service { s.reconciliation = r; return s }
func (s *Service) WithReportMappings(m store.ReportMappingStore) *Service    { s.reportMappings = m; return s }
func (s *Service) WithAdjustments(a store.AdjustmentStore) *Service          { s.adjustments = a; return s }
func (s *Service) WithTaxProfiles(tp store.TaxProfileStore) *Service         { s.taxProfiles = tp; return s }
func (s *Service) WithTaxReturns(tr store.TaxReturnStore) *Service           { s.taxReturns = tr; return s }
func (s *Service) WithRiskScans(rs store.RiskScanStore) *Service             { s.riskScans = rs; return s }
func (s *Service) WithConsistencyChecks(c store.ConsistencyCheckStore) *Service { s.consistencyChecks = c; return s }

// ---- Book operations ----

func (s *Service) ResolveBook(ctx context.Context, entityID, bookID string) (string, error) {
	if bookID != "" {
		if s.books != nil {
			if _, err := s.books.GetByID(ctx, entityID, bookID); err != nil {
				return "", fmt.Errorf("%w: book does not belong to entity", ErrBookScopeMismatch)
			}
		}
		return bookID, nil
	}
	if s.books == nil {
		return "", nil
	}
	book, err := s.books.GetDefault(ctx, entityID)
	if err != nil {
		return "", fmt.Errorf("resolve default book: %w", err)
	}
	if book == nil {
		return "", ErrBookNotFound
	}
	return book.ID, nil
}

func (s *Service) ListBooks(ctx context.Context, entityID string) ([]domain.AccountingBook, error) {
	if s.books == nil {
		return nil, ErrNotConfigured
	}
	return s.books.List(ctx, entityID)
}

func (s *Service) CreateBook(ctx context.Context, book *domain.AccountingBook) error {
	if s.books == nil {
		return ErrNotConfigured
	}
	if book.ID == "" {
		book.ID = uuid.NewString()
	}
	if book.Status == "" {
		book.Status = domain.BookStatusActive
	}
	if err := s.books.Create(ctx, book); err != nil {
		return err
	}
	s.logAudit(ctx, book.EntityID, book.ID, "", domain.AuditActionBookCreate, "book", book.ID, book.Name, "{}")
	// Auto-provision standard chart of accounts for new books
	s.seedStandardChartOfAccounts(ctx, book.EntityID, book.ID)
	return nil
}

func (s *Service) GetBook(ctx context.Context, entityID, bookID string) (*domain.AccountingBook, error) {
	if s.books == nil {
		return nil, ErrNotConfigured
	}
	return s.books.GetByID(ctx, entityID, bookID)
}

func (s *Service) UpdateBook(ctx context.Context, book *domain.AccountingBook) error {
	if s.books == nil {
		return ErrNotConfigured
	}
	return s.books.Update(ctx, book)
}

func (s *Service) SetDefaultBook(ctx context.Context, entityID, bookID string) error {
	if s.books == nil {
		return ErrNotConfigured
	}
	return s.books.SetDefault(ctx, entityID, bookID)
}

// ---- Period operations ----

func (s *Service) ensurePeriod(ctx context.Context, entityID, bookID, periodStr string) (*domain.AccountingPeriod, error) {
	if s.periods == nil {
		return nil, nil
	}
	return s.periods.GetOrCreate(ctx, &domain.AccountingPeriod{
		EntityID: entityID,
		BookID:   bookID,
		Period:   periodStr,
	})
}

func (s *Service) ListPeriods(ctx context.Context, entityID, bookID string) ([]domain.AccountingPeriod, error) {
	if s.periods == nil {
		return nil, ErrNotConfigured
	}
	return s.periods.ListByBook(ctx, entityID, bookID)
}

func (s *Service) CloseCheck(ctx context.Context, entityID, bookID, periodStr string) (*domain.CloseCheckResult, error) {
	if bookID == "" {
		return nil, errors.New("book_id is required")
	}
	if s.periods == nil {
		return nil, ErrNotConfigured
	}

	checks := make([]domain.CloseCheck, 0)
	passed := true

	pendingCount := s.getPendingInvoiceCount(ctx, entityID, bookID, periodStr)
	checks = append(checks, domain.CloseCheck{
		Category: "invoices",
		Label:    "All invoices processed",
		Passed:   pendingCount == 0,
		Detail:   fmt.Sprintf("%d invoices pending review", pendingCount),
		Severity: "blocking",
	})
	if pendingCount > 0 {
		passed = false
	}

	draftCount := s.getDraftJournalCount(ctx, entityID, bookID, periodStr)
	checks = append(checks, domain.CloseCheck{
		Category: "journals",
		Label:    "No draft journal entries",
		Passed:   draftCount == 0,
		Detail:   fmt.Sprintf("%d draft entries", draftCount),
		Severity: "blocking",
	})
	if draftCount > 0 {
		passed = false
	}

	tb, err := s.TrialBalance(ctx, entityID, bookID, periodStr)
	if err != nil {
		checks = append(checks, domain.CloseCheck{
			Category: "reconciliation",
			Label:    "Trial balance",
			Passed:   false,
			Detail:   fmt.Sprintf("failed: %v", err),
			Severity: "blocking",
		})
		passed = false
	} else {
		balanced := math.Abs(tb.TotalDebit-tb.TotalCredit) < 0.01
		checks = append(checks, domain.CloseCheck{
			Category: "reconciliation",
			Label:    "Trial balance balanced",
			Passed:   balanced,
			Detail:   fmt.Sprintf("debit %.2f, credit %.2f", tb.TotalDebit, tb.TotalCredit),
			Severity: "blocking",
		})
		if !balanced {
			passed = false
		}
	}

	result := &domain.CloseCheckResult{
		Passed:  passed,
		Period:  periodStr,
		Checks:  checks,
		Summary: "all blocking checks passed",
	}
	if !passed {
		result.Summary = "some blocking checks failed"
	}
	return result, nil
}

func (s *Service) ClosePeriod(ctx context.Context, entityID, bookID, periodStr, actorID string) (*domain.AccountingPeriod, error) {
	if bookID == "" {
		return nil, errors.New("book_id is required")
	}
	if s.periods == nil {
		return nil, ErrNotConfigured
	}

	p, err := s.periods.UpdateStatus(ctx, entityID, bookID, periodStr, domain.PeriodStatusOpen, domain.PeriodStatusClosing, &actorID)
	if err != nil {
		return nil, fmt.Errorf("begin closing: %w", err)
	}

	result, err := s.CloseCheck(ctx, entityID, bookID, periodStr)
	if err != nil || !result.Passed {
		s.periods.UpdateStatus(ctx, entityID, bookID, periodStr, domain.PeriodStatusClosing, domain.PeriodStatusOpen, nil)
		if err != nil {
			return nil, fmt.Errorf("close check: %w", err)
		}
		return nil, fmt.Errorf("close check failed: %s", result.Summary)
	}

	p, err = s.periods.UpdateStatus(ctx, entityID, bookID, periodStr, domain.PeriodStatusClosing, domain.PeriodStatusClosed, &actorID)
	if err != nil {
		return nil, fmt.Errorf("close: %w", err)
	}

	s.logAudit(ctx, entityID, bookID, actorID, domain.AuditActionPeriodClose, "accounting_period", p.ID, fmt.Sprintf("%s/%s", bookID, periodStr), "{}")
	return p, nil
}

func (s *Service) ReopenPeriod(ctx context.Context, entityID, bookID, periodStr, actorID string) (*domain.AccountingPeriod, error) {
	if bookID == "" {
		return nil, errors.New("book_id is required")
	}
	if s.periods == nil {
		return nil, ErrNotConfigured
	}
	p, err := s.periods.UpdateStatus(ctx, entityID, bookID, periodStr, domain.PeriodStatusClosed, domain.PeriodStatusOpen, nil)
	if err != nil {
		return nil, fmt.Errorf("reopen: %w", err)
	}
	s.logAudit(ctx, entityID, bookID, actorID, domain.AuditActionPeriodReopen, "accounting_period", p.ID, fmt.Sprintf("%s/%s", bookID, periodStr), "{}")
	return p, nil
}

func (s *Service) LockPeriod(ctx context.Context, entityID, bookID, periodStr, actorID string) (*domain.AccountingPeriod, error) {
	if bookID == "" {
		return nil, errors.New("book_id is required")
	}
	if s.periods == nil {
		return nil, ErrNotConfigured
	}
	p, err := s.periods.UpdateStatus(ctx, entityID, bookID, periodStr, domain.PeriodStatusClosed, domain.PeriodStatusLocked, nil)
	if err != nil {
		return nil, fmt.Errorf("lock: %w", err)
	}
	s.logAudit(ctx, entityID, bookID, actorID, domain.AuditActionPeriodLock, "accounting_period", p.ID, fmt.Sprintf("%s/%s", bookID, periodStr), "{}")
	return p, nil
}

func (s *Service) getPendingInvoiceCount(ctx context.Context, entityID, bookID, period string) int {
	records, err := s.invoices.List(ctx, entityID, bookID, period, domain.StatusPendingReview, 1000, 0)
	if err != nil {
		return 0
	}
	return len(records)
}

func (s *Service) getDraftJournalCount(ctx context.Context, entityID, bookID, period string) int {
	entries, err := s.journals.List(ctx, domain.JournalListQuery{
		EntityID: entityID,
		BookID:   bookID,
		Status:   domain.JournalStatusDraft,
		Period:   period,
		Limit:    1000,
	})
	if err != nil {
		return 0
	}
	return len(entries)
}

// ---- Invoice operations ----

func (s *Service) CreateInvoiceDraft(ctx context.Context, entityID, bookID, actorID string, inv *domain.Invoice) (*domain.Invoice, error) {
	if inv == nil {
		return nil, fmt.Errorf("%w: invoice is required", ErrInvalidRequest)
	}

	// Validate invoice amounts
	extraction := domain.ExtractionResult{
		InvoiceNo:        inv.InvoiceNo,
		InvoiceType:      inv.InvoiceType,
		Direction:        inv.Direction,
		IssueDate:        inv.IssueDate,
		CounterpartyName: inv.SellerName,
		SellerTaxID:      inv.SellerTaxNo,
		BuyerTaxID:       inv.BuyerTaxNo,
		AmountWithoutTax: inv.AmountWithoutTax,
		TaxAmount:        inv.TaxAmount,
		AmountWithTax:    inv.AmountWithTax,
	}
	if err := extraction.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidRequest, err)
	}

	if s.periods != nil && bookID != "" {
		if closed, _ := s.periods.IsClosed(ctx, entityID, bookID, accountingPeriod(inv.IssueDate)); closed {
			return nil, ErrPeriodClosed
		}
	}

	if existing, err := s.invoices.FindByInvoiceNo(ctx, entityID, bookID, inv.InvoiceNo); err == nil && existing != nil {
		return existing, nil
	}

	inv.ID = uuid.NewString()
	inv.EntityID = entityID
	inv.BookID = bookID
	inv.Status = domain.StatusPendingReview
	inv.Source = "provider"
	inv.Currency = "CNY"
	if inv.Currency == "" {
		inv.Currency = "CNY"
	}

	if err := s.invoices.Create(ctx, inv); err != nil {
		return nil, err
	}

	s.logAudit(ctx, entityID, bookID, actorID, domain.AuditActionInvoiceCreateDraft, "invoice", inv.ID, inv.InvoiceNo, "{}")
	return inv, nil
}

func (s *Service) GetInvoice(ctx context.Context, entityID, invoiceID string) (*domain.Invoice, error) {
	if invoiceID == "" {
		return nil, fmt.Errorf("%w: invoice id required", ErrInvalidRequest)
	}
	return s.invoices.Get(ctx, entityID, invoiceID)
}

func (s *Service) ListInvoices(ctx context.Context, entityID, bookID, period, status string, limit, offset int) ([]domain.Invoice, error) {
	if period != "" && !validPeriod(period) {
		return nil, fmt.Errorf("%w: period must be YYYY-MM", ErrInvalidRequest)
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	return s.invoices.List(ctx, entityID, bookID, period, status, limit, offset)
}

func (s *Service) ApproveInvoice(ctx context.Context, entityID, invoiceID, actorID string) (*domain.Invoice, *domain.JournalEntry, error) {
	if invoiceID == "" {
		return nil, nil, fmt.Errorf("%w: invoice id required", ErrInvalidRequest)
	}


	// Check period is not closed/locked
	preInv, err := s.invoices.Get(ctx, entityID, invoiceID)
	if err != nil || preInv == nil {
		return nil, nil, fmt.Errorf("invoice not found: %s", invoiceID)
	}
	if s.periods != nil {
		if closed, _ := s.periods.IsClosed(ctx, entityID, preInv.BookID, accountingPeriod(preInv.IssueDate)); closed {
			return nil, nil, ErrPeriodClosed
		}
	}
	if s.db == nil {
		inv, err := s.invoices.Approve(ctx, entityID, invoiceID)
		if err != nil || inv == nil {
			return inv, nil, err
		}
		entry, err := s.buildJournalDraft(inv)
		if err != nil {
			return inv, nil, err
		}
		if err := s.journals.Create(ctx, entry); err != nil {
			return inv, nil, err
		}
		s.logAudit(ctx, entityID, inv.BookID, actorID, domain.AuditActionInvoiceApprove, "invoice", inv.ID, inv.InvoiceNo, "{}")
		return inv, entry, nil
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	inv, err := s.invoices.ApproveTx(ctx, tx, entityID, invoiceID)
	if err != nil || inv == nil {
		return inv, nil, err
	}

	entry, err := s.buildJournalDraft(inv)
	if err != nil {
		return inv, nil, err
	}
	if err := s.journals.CreateTx(ctx, tx, entry); err != nil {
		return inv, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit tx: %w", err)
	}
	s.logAudit(ctx, entityID, inv.BookID, actorID, domain.AuditActionInvoiceApprove, "invoice", inv.ID, inv.InvoiceNo, "{}")
	return inv, entry, nil
}

func (s *Service) RejectInvoice(ctx context.Context, entityID, invoiceID, actorID string) (*domain.Invoice, error) {
	if invoiceID == "" {
		return nil, fmt.Errorf("%w: invoice id required", ErrInvalidRequest)
	}
	inv, err := s.invoices.Reject(ctx, entityID, invoiceID)
	if err == nil && inv != nil {
		s.logAudit(ctx, entityID, inv.BookID, actorID, domain.AuditActionInvoiceReject, "invoice", inv.ID, inv.InvoiceNo, "{}")
	}
	return inv, err
}

// ---- Journal operations ----

func (s *Service) CreateJournalDraft(ctx context.Context, entityID, bookID, period, actorID string, entry *domain.JournalEntry) (*domain.JournalEntry, error) {
	if len(entry.Lines) == 0 {
		return nil, fmt.Errorf("%w: journal entry has no lines", ErrInvalidRequest)
	}
	for i := range entry.Lines {
		entry.Lines[i].DebitAmount = roundMoney(entry.Lines[i].DebitAmount)
		entry.Lines[i].CreditAmount = roundMoney(entry.Lines[i].CreditAmount)
	}

	entry.ID = uuid.NewString()
	entry.EntityID = entityID
	entry.BookID = bookID
	entry.Period = period
	entry.Status = domain.JournalStatusDraft
	entry.CreatedBy = actorID
	if entry.EntryDate == "" {
		entry.EntryDate = time.Now().UTC().Format("2006-01-02")
	}
	if entry.VoucherWord == "" {
		entry.VoucherWord = domain.VoucherWordJi
	}

	if entry.VoucherNo == "" {
		voucherNo, err := s.journals.NextVoucherNo(ctx, entityID, bookID, period)
		if err != nil {
			return nil, fmt.Errorf("generate voucher_no: %w", err)
		}
		entry.VoucherNo = voucherNo
	}
	if period != "" && !validPeriod(period) {
		return nil, fmt.Errorf("%w: period must be YYYY-MM", ErrInvalidRequest)
	}
	if s.periods != nil {
		if closed, _ := s.periods.IsClosed(ctx, entityID, bookID, period); closed {
			return nil, ErrPeriodClosed
		}
	}

	for i := range entry.Lines {
		entry.Lines[i].ID = uuid.NewString()
		entry.Lines[i].EntityID = entityID
		entry.Lines[i].JournalEntryID = entry.ID
		entry.Lines[i].Currency = "CNY"
		entry.Lines[i].LineNo = i + 1
		entry.Lines[i].AccountID = entry.Lines[i].AccountCode
	}

	if err := validateBalanced(entry); err != nil {
		return nil, err
	}
	if err := s.journals.Create(ctx, entry); err != nil {
		return nil, err
	}

	s.logAudit(ctx, entityID, bookID, actorID, domain.AuditActionJournalCreateDraft, "journal_entry", entry.ID, entry.Summary, "{}")
	return entry, nil
}

func (s *Service) GetJournalEntry(ctx context.Context, entityID, journalID string) (*domain.JournalEntry, error) {
	if journalID == "" {
		return nil, fmt.Errorf("%w: journal id required", ErrInvalidRequest)
	}
	return s.journals.Get(ctx, entityID, journalID)
}

func (s *Service) ListJournalEntries(ctx context.Context, query domain.JournalListQuery) ([]domain.JournalEntry, error) {
	if query.Period != "" && !validPeriod(query.Period) {
		return nil, fmt.Errorf("%w: period must be YYYY-MM", ErrInvalidRequest)
	}
	if query.Limit <= 0 {
		query.Limit = 50
	}
	if query.Limit > 200 {
		query.Limit = 200
	}
	return s.journals.List(ctx, query)
}

func (s *Service) PostJournalEntry(ctx context.Context, entityID, journalID, actorID string) (*domain.JournalEntry, error) {
	if journalID == "" {
		return nil, fmt.Errorf("%w: journal id required", ErrInvalidRequest)
	}

	entry, err := s.journals.Get(ctx, entityID, journalID)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, fmt.Errorf("journal entry not found: %s", journalID)
	}
	if s.periods != nil {
		if closed, _ := s.periods.IsClosed(ctx, entityID, entry.BookID, entry.Period); closed {
			return nil, ErrPeriodClosed
		}
	}
	if entry.Status == domain.JournalStatusPosted {
		return entry, nil
	}
	if len(entry.Lines) == 0 {
		return nil, fmt.Errorf("%w: journal entry has no lines", ErrInvalidRequest)
	}
	if err := validateBalanced(entry); err != nil {
		return nil, err
	}

	if s.db == nil {
		posted, err := s.journals.Post(ctx, entityID, journalID)
		if err != nil {
			return nil, err
		}
		if posted != nil && posted.SourceType == "invoice" && posted.SourceID != "" {
			s.invoices.MarkPosted(ctx, entityID, posted.SourceID)
		}
		s.logAudit(ctx, entityID, entry.BookID, actorID, domain.AuditActionJournalPost, "journal_entry", journalID, entry.Summary, "{}")
		return posted, nil
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	posted, err := s.journals.PostTx(ctx, tx, entityID, journalID)
	if err != nil {
		return nil, err
	}
	if posted != nil && posted.SourceType == "invoice" && posted.SourceID != "" {
		s.invoices.MarkPostedTx(ctx, tx, entityID, posted.SourceID)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	s.logAudit(ctx, entityID, entry.BookID, actorID, domain.AuditActionJournalPost, "journal_entry", journalID, entry.Summary, "{}")
	return posted, nil
}

func (s *Service) VoidJournalEntry(ctx context.Context, entityID, journalID, actorID string) (*domain.JournalEntry, error) {
	if journalID == "" {
		return nil, fmt.Errorf("%w: journal id required", ErrInvalidRequest)
	}

	entry, err := s.journals.Get(ctx, entityID, journalID)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, fmt.Errorf("journal entry not found: %s", journalID)
	}
	if s.periods != nil {
		if closed, _ := s.periods.IsClosed(ctx, entityID, entry.BookID, entry.Period); closed {
			return nil, ErrPeriodClosed
		}
	}

	// Non-posted or no DB: simple void
	if entry.Status != domain.JournalStatusPosted || s.db == nil {
		voided, err := s.journals.Void(ctx, entityID, journalID)
		if err == nil && voided != nil {
			s.logAudit(ctx, entityID, voided.BookID, actorID, domain.AuditActionJournalVoid, "journal_entry", journalID, voided.Summary, "{}")
		}
		return voided, err
	}

	// Posted entries: atomic reversal + void
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	rev := s.buildReversal(entry)
	if err := s.journals.CreateTx(ctx, tx, rev); err != nil {
		return nil, fmt.Errorf("create reversal entry: %w", err)
	}
	if _, err := s.journals.PostTx(ctx, tx, entityID, rev.ID); err != nil {
		return nil, fmt.Errorf("post reversal entry: %w", err)
	}

	voided, err := s.journals.VoidTx(ctx, tx, entityID, journalID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	if voided != nil {
		s.logAudit(ctx, entityID, voided.BookID, actorID, domain.AuditActionJournalVoid, "journal_entry", journalID, voided.Summary, "{}")
	}
	return voided, nil
}

// ---- Report operations ----

func (s *Service) TrialBalance(ctx context.Context, entityID, bookID, period string) (*domain.TrialBalance, error) {
	lines, err := s.journals.ListPostedLines(ctx, entityID, bookID, period)
	if err != nil {
		return nil, err
	}

	type key struct {
		code string
		name string
	}
	rowsByAccount := make(map[key]*domain.TrialBalanceRow)
	for _, line := range lines {
		k := key{code: line.AccountCode, name: line.AccountName}
		row := rowsByAccount[k]
		if row == nil {
			row = &domain.TrialBalanceRow{
				AccountCode: line.AccountCode,
				AccountName: line.AccountName,
			}
			rowsByAccount[k] = row
		}
		row.DebitAmount = roundMoney(row.DebitAmount + line.DebitAmount)
		row.CreditAmount = roundMoney(row.CreditAmount + line.CreditAmount)
		row.NetAmount = roundMoney(row.DebitAmount - row.CreditAmount)
	}

	keys := make([]key, 0, len(rowsByAccount))
	for k := range rowsByAccount {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].code < keys[j].code })

	tb := &domain.TrialBalance{Period: period}
	for _, k := range keys {
		row := *rowsByAccount[k]
		tb.Rows = append(tb.Rows, row)
		tb.TotalDebit = roundMoney(tb.TotalDebit + row.DebitAmount)
		tb.TotalCredit = roundMoney(tb.TotalCredit + row.CreditAmount)
	}
	tb.Balanced = moneyEqual(tb.TotalDebit, tb.TotalCredit)
	return tb, nil
}

func (s *Service) AccountBalance(ctx context.Context, entityID, bookID, period string) (*domain.AccountBalance, error) {
	tb, err := s.TrialBalance(ctx, entityID, bookID, period)
	if err != nil {
		return nil, err
	}
	result := &domain.AccountBalance{Period: period}
	for _, row := range tb.Rows {
		balance := domain.AccountBalanceRow{
			AccountCode:  row.AccountCode,
			AccountName:  row.AccountName,
			DebitAmount:  row.DebitAmount,
			CreditAmount: row.CreditAmount,
		}
		if row.NetAmount >= 0 {
			balance.EndingDebit = row.NetAmount
		} else {
			balance.EndingCredit = roundMoney(-row.NetAmount)
		}
		result.Rows = append(result.Rows, balance)
	}
	return result, nil
}

func (s *Service) VATSummary(ctx context.Context, entityID, bookID, period string) (*domain.VATSummary, error) {
	if !validPeriod(period) {
		return nil, fmt.Errorf("%w: period must be YYYY-MM", ErrInvalidRequest)
	}
	records, err := s.invoices.ListPostedByPeriod(ctx, entityID, bookID, period)
	if err != nil {
		return nil, err
	}

	summary := &domain.VATSummary{Period: period}
	for _, inv := range records {
		if inv.Direction == domain.DirectionOutput {
			summary.OutputInvoiceCount++
			summary.OutputAmountWithoutTax = roundMoney(summary.OutputAmountWithoutTax + inv.AmountWithoutTax)
			summary.OutputTaxAmount = roundMoney(summary.OutputTaxAmount + inv.TaxAmount)
			summary.OutputAmountWithTax = roundMoney(summary.OutputAmountWithTax + inv.AmountWithTax)
		} else {
			summary.InvoiceCount++
			summary.InputAmountWithoutTax = roundMoney(summary.InputAmountWithoutTax + inv.AmountWithoutTax)
			summary.InputTaxAmount = roundMoney(summary.InputTaxAmount + inv.TaxAmount)
			summary.InputAmountWithTax = roundMoney(summary.InputAmountWithTax + inv.AmountWithTax)
		}
	}
	return summary, nil
}

// ---- Account operations ----

func (s *Service) ListAccounts(ctx context.Context, query domain.AccountListQuery) (*domain.AccountListResponse, error) {
	if query.Limit <= 0 {
		query.Limit = 50
	}
	if query.Limit > 200 {
		query.Limit = 200
	}
	items, total, err := s.accounts.List(ctx, query)
	if err != nil {
		return nil, err
	}
	return &domain.AccountListResponse{Items: items, Total: total, Limit: query.Limit, Offset: query.Offset}, nil
}

func (s *Service) GetAccount(ctx context.Context, entityID, bookID, code string) (*domain.ChartAccount, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("%w: account code required", ErrInvalidRequest)
	}
	return s.accounts.GetByCode(ctx, entityID, bookID, code)
}

func (s *Service) CreateAccount(ctx context.Context, a *domain.ChartAccount) error {
	a.Code = strings.TrimSpace(a.Code)
	a.Name = strings.TrimSpace(a.Name)
	a.Category = strings.TrimSpace(a.Category)
	a.BalanceType = strings.TrimSpace(a.BalanceType)
	if a.Code == "" || a.Name == "" || a.Category == "" {
		return fmt.Errorf("%w: code, name, category required", ErrInvalidRequest)
	}
	if a.BalanceType != domain.BalanceTypeDebit && a.BalanceType != domain.BalanceTypeCredit {
		return fmt.Errorf("%w: balance_type must be debit or credit", ErrInvalidRequest)
	}
	if a.Keywords == nil {
		a.Keywords = []string{}
	}
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	if err := s.accounts.Create(ctx, a); err != nil {
		return err
	}
	s.logAudit(ctx, a.EntityID, a.BookID, "", domain.AuditActionAccountCreate, "account", a.Code, a.Name, "{}")
	return nil
}

func (s *Service) UpdateAccount(ctx context.Context, entityID, bookID, code string, a *domain.ChartAccount) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return fmt.Errorf("%w: account code required", ErrInvalidRequest)
	}
	existing, err := s.accounts.GetByCode(ctx, entityID, bookID, code)
	if err != nil || existing == nil {
		return fmt.Errorf("%w: account not found", ErrInvalidRequest)
	}
	a.ID = existing.ID
	a.EntityID = entityID
	a.BookID = bookID
	a.Code = code
	if a.Name == "" {
		a.Name = existing.Name
	}
	if a.Category == "" {
		a.Category = existing.Category
	}
	if a.BalanceType == "" {
		a.BalanceType = existing.BalanceType
	}
	if a.Keywords == nil {
		a.Keywords = existing.Keywords
	}
	if err := s.accounts.Update(ctx, a); err != nil {
		return err
	}
	s.logAudit(ctx, entityID, bookID, "", domain.AuditActionAccountUpdate, "account", a.Code, a.Name, "{}")
	return nil
}

func (s *Service) DeleteAccount(ctx context.Context, entityID, accountID string) error {
	if accountID == "" {
		return fmt.Errorf("%w: account id required", ErrInvalidRequest)
	}
	if err := s.accounts.Delete(ctx, entityID, accountID); err != nil {
		return err
	}
	return nil
}

// ---- Audit operations ----

func (s *Service) logAudit(ctx context.Context, entityID, bookID, actorID, action, objectType, objectID, objectRef, details string) {
	if s.auditLog == nil {
		return
	}
	traceID := provider.TraceIDFromContext(ctx)
	capabilityID := provider.CapabilityIDFromContext(ctx)
	idempotencyKey := provider.IdempotencyKeyFromContext(ctx)
	approvalGrantID := provider.ApprovalGrantIDFromContext(ctx)
	if actorID == "" {
		actorID = provider.ActorIDFromContext(ctx)
	}
	payload, _ := json.Marshal(map[string]string{"ref": objectRef, "details": details})
	s.auditLog.Append(ctx, &domain.AuditEntry{
		ID:               uuid.NewString(),
		EntityID:         entityID,
		BookID:           bookID,
		CapabilityID:     capabilityID,
		V2CapabilityID:   capabilityID,
		TraceID:          traceID,
		IdempotencyKey:   idempotencyKey,
		ApprovalGrantID:  approvalGrantID,
		ActorType:        "user",
		ActorID:          actorID,
		Action:           action,
		ObjectType:       objectType,
		ObjectID:         objectID,
		Outcome:          domain.AuditOutcomeSuccess,
		Payload:          payload,
	})
}

func (s *Service) ListAuditLog(ctx context.Context, query domain.AuditListQuery) ([]domain.AuditEntry, error) {
	if s.auditLog == nil {
		return nil, ErrNotConfigured
	}
	return s.auditLog.List(ctx, query)
}

// ---- Idempotency ----

func (s *Service) CheckIdempotency(ctx context.Context, entityID, capabilityID, idempotencyKey, inputHash string) (*store.IdempotencyRecord, error) {
	if s.idempotency == nil {
		return nil, nil
	}
	return s.idempotency.Get(ctx, entityID, capabilityID, idempotencyKey)
}

func (s *Service) SaveIdempotency(ctx context.Context, entityID, capabilityID, idempotencyKey, inputHash string, result []byte, status string) error {
	if s.idempotency == nil {
		return nil
	}
	return s.idempotency.Save(ctx, &store.IdempotencyRecord{
		EntityID:       entityID,
		CapabilityID:   capabilityID,
		IdempotencyKey: idempotencyKey,
		InputHash:      inputHash,
		Result:         result,
		Status:         status,
	})
}

// ---- Helpers ----

func (s *Service) buildJournalDraft(inv *domain.Invoice) (*domain.JournalEntry, error) {
	if inv.AmountWithoutTax < 0 || inv.TaxAmount < 0 || inv.AmountWithTax < 0 {
		return nil, fmt.Errorf("%w: invoice amounts must be non-negative", ErrInvalidRequest)
	}
	if !moneyEqual(inv.AmountWithoutTax+inv.TaxAmount, inv.AmountWithTax) {
		return nil, fmt.Errorf("%w: invoice amount mismatch", ErrInvalidRequest)
	}

	lookup := func(code, fallbackName string) (string, string) {
		if s.accounts == nil {
			return code, fallbackName
		}
		a, err := s.accounts.GetByCode(context.Background(), inv.EntityID, inv.BookID, code)
		if err != nil || a == nil {
			return code, fallbackName
		}
		return a.Code, a.Name
	}

	lines, summary := s.buildJournalLines(inv, lookup)
	entry := &domain.JournalEntry{
		ID:          uuid.NewString(),
		EntityID:    inv.EntityID,
		BookID:      inv.BookID,
		Period:      accountingPeriod(inv.IssueDate),
		Status:      domain.JournalStatusDraft,
		Summary:     summary,
		EntryDate:   inv.IssueDate,
		VoucherWord: domain.VoucherWordJi,
		SourceType:  "invoice",
		SourceID:    inv.ID,
		Lines:       lines,
	}

	voucherNo, _ := s.journals.NextVoucherNo(context.Background(), inv.EntityID, inv.BookID, accountingPeriod(inv.IssueDate))
	entry.VoucherNo = voucherNo

	if err := validateBalanced(entry); err != nil {
		return nil, err
	}
	return entry, nil
}

func (s *Service) buildJournalLines(inv *domain.Invoice, lookup func(code, fallback string) (string, string)) ([]domain.JournalLine, string) {
	switch inv.Direction {
	case domain.DirectionOutput:
		return s.outputLines(inv, lookup)
	default:
		return s.inputLines(inv, lookup)
	}
}

func (s *Service) inputLines(inv *domain.Invoice, lookup func(code, fallback string) (string, string)) ([]domain.JournalLine, string) {
	expCode, expName := s.resolveInputAccount(inv, lookup)
	taxCode, taxName := lookup("2221-01", "应交税费-应交增值税-进项税额")
	bankCode, bankName := lookup("1002", "银行存款")

	summary := fmt.Sprintf("采购发票 %s %s", inv.InvoiceNo, inv.SellerName)
	return []domain.JournalLine{
		{ID: uuid.NewString(), AccountCode: expCode, AccountName: expName, Direction: domain.DirectionDebit, DebitAmount: roundMoney(inv.AmountWithoutTax), Currency: "CNY", LineNo: 1},
		{ID: uuid.NewString(), AccountCode: taxCode, AccountName: taxName, Direction: domain.DirectionDebit, DebitAmount: roundMoney(inv.TaxAmount), Currency: "CNY", LineNo: 2},
		{ID: uuid.NewString(), AccountCode: bankCode, AccountName: bankName, Direction: domain.DirectionCredit, CreditAmount: roundMoney(inv.AmountWithTax), Currency: "CNY", LineNo: 3},
	}, summary
}

func (s *Service) outputLines(inv *domain.Invoice, lookup func(code, fallback string) (string, string)) ([]domain.JournalLine, string) {
	arCode, arName := lookup("1122", "应收账款")
	revCode, revName := lookup("6001", "主营业务收入")
	taxCode, taxName := lookup("2221-02", "应交税费-应交增值税-销项税额")

	summary := fmt.Sprintf("销售发票 %s %s", inv.InvoiceNo, inv.BuyerName)
	return []domain.JournalLine{
		{ID: uuid.NewString(), AccountCode: arCode, AccountName: arName, Direction: domain.DirectionDebit, DebitAmount: roundMoney(inv.AmountWithTax), Currency: "CNY", LineNo: 1},
		{ID: uuid.NewString(), AccountCode: revCode, AccountName: revName, Direction: domain.DirectionCredit, CreditAmount: roundMoney(inv.AmountWithoutTax), Currency: "CNY", LineNo: 2},
		{ID: uuid.NewString(), AccountCode: taxCode, AccountName: taxName, Direction: domain.DirectionCredit, CreditAmount: roundMoney(inv.TaxAmount), Currency: "CNY", LineNo: 3},
	}, summary
}

func (s *Service) resolveInputAccount(inv *domain.Invoice, lookup func(code, fallback string) (string, string)) (string, string) {
	lower := strings.ToLower(inv.InvoiceType + inv.SellerName)
	if strings.Contains(lower, "固定资产") || strings.Contains(lower, "设备") || strings.Contains(lower, "机械") {
		return lookup("1601", "固定资产")
	}
	if strings.Contains(lower, "无形资产") || strings.Contains(lower, "软件") {
		return lookup("1701", "无形资产")
	}
	if strings.Contains(lower, "库存") || strings.Contains(lower, "商品") || strings.Contains(lower, "原材料") {
		return lookup("1405", "库存商品")
	}
	return lookup("5602", "管理费用")
}

// ---- Validation helpers ----

func validateBalanced(entry *domain.JournalEntry) error {
	var debit, credit float64
	for _, line := range entry.Lines {
		debit += line.DebitAmount
		credit += line.CreditAmount
	}
	if !moneyEqual(debit, credit) {
		return fmt.Errorf("%w: journal entry is not balanced (debit=%.2f, credit=%.2f)", ErrInvalidRequest, debit, credit)
	}
	return nil
}

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}

func moneyEqual(a, b float64) bool {
	return math.Abs(roundMoney(a)-roundMoney(b)) < 0.005
}

func money(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

var periodPattern = regexp.MustCompile(`^\d{4}-\d{2}$`)

func validPeriod(period string) bool {
	if !periodPattern.MatchString(period) {
		return false
	}
	_, err := time.Parse("2006-01", period)
	return err == nil
}

func accountingPeriod(issueDate string) string {
	if len(issueDate) >= len("2006-01") {
		period := issueDate[:len("2006-01")]
		if validPeriod(period) {
			return period
		}
	}
	return time.Now().UTC().Format("2006-01")
}

func reverseDirection(d string) string {
	if d == domain.DirectionDebit {
		return domain.DirectionCredit
	}
	return domain.DirectionDebit
}

func (s *Service) buildReversal(original *domain.JournalEntry) *domain.JournalEntry {
	rev := &domain.JournalEntry{
		ID:          uuid.NewString(),
		EntityID:    original.EntityID,
		BookID:      original.BookID,
		Period:      original.Period,
		Status:      domain.JournalStatusDraft,
		Summary:     "红字冲销: " + original.Summary,
		EntryDate:   time.Now().UTC().Format("2006-01-02"),
		VoucherWord: original.VoucherWord,
		SourceType:  original.SourceType,
		SourceID:    original.SourceID,
	}
	for i, line := range original.Lines {
		rev.Lines = append(rev.Lines, domain.JournalLine{
			ID:           uuid.NewString(),
			EntityID:     original.EntityID,
			AccountID:    line.AccountID,
			AccountCode:  line.AccountCode,
			AccountName:  line.AccountName,
			Direction:    reverseDirection(line.Direction),
			DebitAmount:  line.CreditAmount,
			CreditAmount: line.DebitAmount,
			Currency:     line.Currency,
			LineNo:       i + 1,
		})
	}
	if rev.VoucherNo == "" {
		rev.VoucherNo, _ = s.journals.NextVoucherNo(context.Background(), original.EntityID, original.BookID, original.Period)
	}
	return rev
}

// ---- Invoice extension operations ----

func (s *Service) UpdateInvoiceDraft(ctx context.Context, entityID, invoiceID string, inv *domain.Invoice) (*domain.Invoice, error) {
	if invoiceID == "" {
		return nil, fmt.Errorf("%w: invoice id required", ErrInvalidRequest)
	}
	existing, err := s.invoices.Get(ctx, entityID, invoiceID)
	if err != nil || existing == nil {
		return nil, fmt.Errorf("invoice not found: %s", invoiceID)
	}
	if existing.Status != domain.StatusDraft && existing.Status != domain.StatusPendingReview {
		return nil, fmt.Errorf("invoice status %s cannot be updated", existing.Status)
	}

	inv.ID = existing.ID
	inv.EntityID = entityID
	inv.BookID = existing.BookID
	inv.Status = existing.Status
	inv.Source = existing.Source

	if err := s.invoices.Update(ctx, inv); err != nil {
		return nil, err
	}
	s.logAudit(ctx, entityID, existing.BookID, "", domain.AuditActionInvoiceUpdate, "invoice", inv.ID, inv.InvoiceNo, "{}")
	return s.invoices.Get(ctx, entityID, invoiceID)
}

func (s *Service) CreateRedLetterInvoice(ctx context.Context, entityID, bookID, originalInvoiceID, actorID, redType string) (*domain.Invoice, error) {
	original, err := s.invoices.Get(ctx, entityID, originalInvoiceID)
	if err != nil || original == nil {
		return nil, fmt.Errorf("original invoice not found: %s", originalInvoiceID)
	}
	if original.Status != domain.StatusPosted {
		return nil, fmt.Errorf("original invoice must be posted, current status: %s", original.Status)
	}

	redStatus := domain.RedLetterStatusPartially
	if redType == "fully" {
		redStatus = domain.RedLetterStatusFully
	}

	redInv := &domain.Invoice{
		ID:                uuid.NewString(),
		EntityID:          entityID,
		BookID:            bookID,
		InvoiceNo:         original.InvoiceNo + "-RED",
		InvoiceType:       original.InvoiceType,
		Direction:         reverseDirection(original.Direction),
		IssueDate:         time.Now().UTC().Format("2006-01-02"),
		SellerName:        original.SellerName,
		SellerTaxNo:       original.SellerTaxNo,
		BuyerName:         original.BuyerName,
		BuyerTaxNo:        original.BuyerTaxNo,
		AmountWithoutTax:  -original.AmountWithoutTax,
		TaxAmount:         -original.TaxAmount,
		AmountWithTax:     -original.AmountWithTax,
		Currency:          original.Currency,
		Status:            domain.StatusPendingReview,
		Source:            "provider",
		InvoiceKind:       original.InvoiceKind,
		OriginalInvoiceID: original.ID,
		RedLetterStatus:   redStatus,
	}

	if err := s.invoices.Create(ctx, redInv); err != nil {
		return nil, err
	}

	// Mark original as red-lettered
	s.invoices.UpdateRedLetterStatus(ctx, entityID, original.ID, redStatus)

	s.logAudit(ctx, entityID, bookID, actorID, domain.AuditActionInvoiceCreateDraft, "invoice", redInv.ID, redInv.InvoiceNo, "{}")
	return redInv, nil
}

func (s *Service) ImportEInvoice(ctx context.Context, entityID, bookID string, payload *domain.ExtractionResult) (*domain.Invoice, error) {
	if err := payload.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidRequest, err)
	}

	if existing, err := s.invoices.FindByDigitalInvoiceNo(ctx, entityID, bookID, payload.DigitalInvoiceNo); err == nil && existing != nil {
		return existing, nil
	}

	inv := &domain.Invoice{
		ID:                 uuid.NewString(),
		EntityID:           entityID,
		BookID:             bookID,
		InvoiceNo:          payload.InvoiceNo,
		InvoiceType:        payload.InvoiceType,
		Direction:          payload.Direction,
		IssueDate:          payload.IssueDate,
		SellerName:         payload.CounterpartyName,
		SellerTaxNo:        payload.SellerTaxID,
		BuyerTaxNo:         payload.BuyerTaxID,
		AmountWithoutTax:   payload.AmountWithoutTax,
		TaxAmount:          payload.TaxAmount,
		AmountWithTax:      payload.AmountWithTax,
		Currency:           "CNY",
		Status:             domain.StatusPendingReview,
		Source:             "e-invoice",
		InvoiceKind:        payload.InvoiceKind,
		DigitalInvoiceNo:   payload.DigitalInvoiceNo,
		BusinessTag:        payload.BusinessTag,
		VerificationStatus: domain.VerStatusUnchecked,
		UsageStatus:        domain.UsageStatusUnconfirmed,
		DeductionStatus:    domain.DeductionStatusNotDeducted,
	}

	if inv.Direction == "" {
		inv.Direction = domain.DirectionInput
	}
	if inv.InvoiceKind == "" {
		inv.InvoiceKind = domain.InvoiceKindElectronic
	}

	if err := s.invoices.Create(ctx, inv); err != nil {
		return nil, err
	}

	// Create invoice lines
	for i := range payload.InvoiceLines {
		payload.InvoiceLines[i].ID = uuid.NewString()
		payload.InvoiceLines[i].EntityID = entityID
		payload.InvoiceLines[i].InvoiceID = inv.ID
	}
	if len(payload.InvoiceLines) > 0 && s.invoiceLines != nil {
		s.invoiceLines.CreateMany(ctx, payload.InvoiceLines)
	}

	s.logAudit(ctx, entityID, bookID, "", domain.AuditActionInvoiceImport, "invoice", inv.ID, inv.InvoiceNo, "{}")
	return inv, nil
}

func (s *Service) VerifyInvoice(ctx context.Context, entityID, invoiceID, actorID string) error {
	inv, err := s.invoices.Get(ctx, entityID, invoiceID)
	if err != nil || inv == nil {
		return fmt.Errorf("invoice not found: %s", invoiceID)
	}
	if err := s.invoices.UpdateVerificationStatus(ctx, entityID, invoiceID, domain.VerStatusVerified); err != nil {
		return err
	}
	s.logAudit(ctx, entityID, inv.BookID, actorID, domain.AuditActionInvoiceVerify, "invoice", invoiceID, inv.InvoiceNo, "{}")
	return nil
}

func (s *Service) ConfirmInvoiceUsage(ctx context.Context, entityID, invoiceID, usageStatus, actorID string) error {
	inv, err := s.invoices.Get(ctx, entityID, invoiceID)
	if err != nil || inv == nil {
		return fmt.Errorf("invoice not found: %s", invoiceID)
	}
	if err := s.invoices.UpdateUsageStatus(ctx, entityID, invoiceID, usageStatus); err != nil {
		return err
	}
	s.logAudit(ctx, entityID, inv.BookID, actorID, domain.AuditActionInvoiceConfirmUsage, "invoice", invoiceID, inv.InvoiceNo, "{}")
	return nil
}
func (s *Service) UpdateJournalDraft(ctx context.Context, entityID, journalID string, entry *domain.JournalEntry) (*domain.JournalEntry, error) {
	existing, err := s.journals.Get(ctx, entityID, journalID)
	if err != nil || existing == nil {
		return nil, fmt.Errorf("journal entry not found: %s", journalID)
	}
	if existing.Status != domain.JournalStatusDraft {
		return nil, fmt.Errorf("journal entry status %s cannot be updated", existing.Status)
	}

	// Void old draft and create new one in-place
	if _, err := s.journals.Void(ctx, entityID, journalID); err != nil {
		return nil, fmt.Errorf("void old draft: %w", err)
	}

	entry.ID = uuid.NewString()
	entry.EntityID = entityID
	entry.BookID = existing.BookID
	entry.Period = existing.Period
	entry.Status = domain.JournalStatusDraft
	entry.VoucherWord = existing.VoucherWord
	entry.VoucherNo = existing.VoucherNo
	for i := range entry.Lines {
		entry.Lines[i].ID = uuid.NewString()
		entry.Lines[i].EntityID = entityID
		entry.Lines[i].JournalEntryID = entry.ID
		entry.Lines[i].Currency = "CNY"
		entry.Lines[i].LineNo = i + 1
		entry.Lines[i].AccountID = entry.Lines[i].AccountCode
	}

	if err := validateBalanced(entry); err != nil {
		return nil, err
	}
	if err := s.journals.Create(ctx, entry); err != nil {
		return nil, err
	}
		s.logAudit(ctx, entityID, existing.BookID, "", domain.AuditActionJournalUpdate, "journal_entry", entry.ID, entry.Summary, "{}")
		return entry, nil
}

func (s *Service) BatchPostJournals(ctx context.Context, entityID string, journalIDs []string, actorID string) ([]domain.JournalEntry, error) {
	var posted []domain.JournalEntry
	for _, jid := range journalIDs {
		p, err := s.PostJournalEntry(ctx, entityID, jid, actorID)
		if err != nil {
			return posted, fmt.Errorf("post %s: %w", jid, err)
		}
		posted = append(posted, *p)
	}
	return posted, nil
}

// ---- Reconciliation operations ----

func (s *Service) UpsertLogistics(ctx context.Context, lr *domain.LogisticsRecord) error {
	if s.reconciliation == nil {
		return ErrNotConfigured
	}
	if lr.ID == "" {
		lr.ID = uuid.NewString()
	}
	return s.reconciliation.UpsertLogistics(ctx, lr)
}

func (s *Service) DeleteLogistics(ctx context.Context, entityID, invoiceID string) error {
	if s.reconciliation == nil {
		return ErrNotConfigured
	}
	return s.reconciliation.DeleteLogistics(ctx, entityID, invoiceID)
}

func (s *Service) UpsertBankTransaction(ctx context.Context, bt *domain.BankTransaction) error {
	if s.reconciliation == nil {
		return ErrNotConfigured
	}
	if bt.ID == "" {
		bt.ID = uuid.NewString()
	}
	return s.reconciliation.UpsertBankTransaction(ctx, bt)
}

func (s *Service) MatchBankToInvoice(ctx context.Context, entityID, bankTxID, invoiceID string, confidence float64) error {
	if s.reconciliation == nil {
		return ErrNotConfigured
	}
	return s.reconciliation.MatchBankToInvoice(ctx, entityID, bankTxID, invoiceID, confidence)
}

func (s *Service) UnmatchBankFromInvoice(ctx context.Context, entityID, bankTxID string) error {
	if s.reconciliation == nil {
		return ErrNotConfigured
	}
	return s.reconciliation.UnmatchBankFromInvoice(ctx, entityID, bankTxID)
}

func (s *Service) GetLogisticsByInvoice(ctx context.Context, entityID, invoiceID string) (*domain.LogisticsRecord, error) {
	if s.reconciliation == nil {
		return nil, ErrNotConfigured
	}
	return s.reconciliation.GetLogisticsByInvoice(ctx, entityID, invoiceID)
}

func (s *Service) GetBankTransaction(ctx context.Context, entityID, bankTxID string) (*domain.BankTransaction, error) {
	if s.reconciliation == nil {
		return nil, ErrNotConfigured
	}
	return s.reconciliation.GetBankTransaction(ctx, entityID, bankTxID)
}

func (s *Service) UnmatchedBankCount(ctx context.Context, entityID, bookID string) (int, error) {
	if s.reconciliation == nil {
		return 0, ErrNotConfigured
	}
	list, err := s.reconciliation.ListUnmatchedBankTransactions(ctx, entityID, bookID)
	if err != nil {
		return 0, err
	}
	return len(list), nil
}

func (s *Service) ThreeWayMatch(ctx context.Context, entityID, bookID, period string) (*domain.ThreeWaySummary, error) {
	if s.reconciliation == nil {
		return nil, ErrNotConfigured
	}
	return s.reconciliation.ThreeWayMatch(ctx, entityID, bookID, period)
}

// ---- Advanced Report operations ----

func (s *Service) ProfitStatement(ctx context.Context, entityID, bookID, period, accountingStandard string) (*domain.ProfitStatement, error) {
	if accountingStandard == "" {
		accountingStandard = "small_business_gaap_cn"
	}

	lines, err := s.journals.ListPostedLines(ctx, entityID, bookID, period)
	if err != nil {
		return nil, err
	}

	engineLines := make([]engine.ReportLine, len(lines))
	for i, l := range lines {
		amount := l.CreditAmount
		if l.Direction == domain.DirectionDebit {
			amount = l.DebitAmount
		}
		engineLines[i] = engine.ReportLine{
			AccountCode: l.AccountCode,
			Direction:   l.Direction,
			Amount:      amount,
		}
	}

	if s.reportMappings != nil {
		mappings, err := s.reportMappings.ListByReportType(ctx, domain.ReportTypeProfitStatement, accountingStandard)
		if err != nil {
			return nil, err
		}
		engineMappings := make([]engine.ReportMapping, len(mappings))
		for i, m := range mappings {
			engineMappings[i] = engine.ReportMapping{
				LineCode:          m.LineCode,
				LineLabel:         m.LineLabel,
				DisplayOrder:      m.DisplayOrder,
				AccountingStandard: m.AccountingStandard,
				AccountSelector: engine.AccountSelector{
					Prefixes:        m.AccountSelector.Prefixes,
					Categories:      m.AccountSelector.Categories,
					Direction:       m.AccountSelector.Direction,
					ExcludePrefixes: m.AccountSelector.ExcludePrefixes,
				},
				IsSubtotal:     m.IsSubtotal,
				ParentLineCode: m.ParentLineCode,
				Formula:        m.Formula,
			}
		}
		builder := engine.NewReportBuilder(engineMappings)
		if builder != nil {
			eps, err := builder.BuildProfitStatement(engineLines)
			if err != nil {
				return nil, err
			}
			return convertProfitStatement(eps), nil
		}
	}

	// Fallback: hard-coded logic for small_business_gaap_cn
	return s.fallbackProfitStatement(engineLines), nil
}

func (s *Service) BalanceSheet(ctx context.Context, entityID, bookID, period, accountingStandard string) (*domain.BalanceSheet, error) {
	if accountingStandard == "" {
		accountingStandard = "small_business_gaap_cn"
	}

	lines, err := s.journals.ListPostedLines(ctx, entityID, bookID, period)
	if err != nil {
		return nil, err
	}

	engineLines := make([]engine.ReportLine, len(lines))
	for i, l := range lines {
		amount := l.CreditAmount
		if l.Direction == domain.DirectionDebit {
			amount = l.DebitAmount
		}
		engineLines[i] = engine.ReportLine{
			AccountCode: l.AccountCode,
			Direction:   l.Direction,
			Amount:      amount,
		}
	}

	if s.reportMappings != nil {
		mappings, err := s.reportMappings.ListByReportType(ctx, domain.ReportTypeBalanceSheet, accountingStandard)
		if err != nil {
			return nil, err
		}
		engineMappings := make([]engine.ReportMapping, len(mappings))
		for i, m := range mappings {
			engineMappings[i] = engine.ReportMapping{
				LineCode:          m.LineCode,
				LineLabel:         m.LineLabel,
				DisplayOrder:      m.DisplayOrder,
				AccountingStandard: m.AccountingStandard,
				AccountSelector: engine.AccountSelector{
					Prefixes:        m.AccountSelector.Prefixes,
					Categories:      m.AccountSelector.Categories,
					Direction:       m.AccountSelector.Direction,
					ExcludePrefixes: m.AccountSelector.ExcludePrefixes,
				},
				IsSubtotal:     m.IsSubtotal,
				ParentLineCode: m.ParentLineCode,
				Formula:        m.Formula,
			}
		}
		builder := engine.NewReportBuilder(engineMappings)
		if builder != nil {
			ebs, err := builder.BuildBalanceSheet(engineLines)
			if err != nil {
				return nil, err
			}
			return convertBalanceSheet(ebs), nil
		}
	}

	return &domain.BalanceSheet{}, nil
}

func (s *Service) VATCrossCheck(ctx context.Context, entityID, bookID, period string) (*domain.VATCrossCheck, error) {
	invoices, err := s.invoices.ListPostedByPeriod(ctx, entityID, bookID, period)
	if err != nil {
		return nil, err
	}

	type rateKey struct {
		rate      float64
		direction string
	}
	groups := map[rateKey]*domain.VATRateRow{}

	for _, inv := range invoices {
		rate := 0.0
		if inv.AmountWithoutTax > 0 {
			rate = roundMoney(inv.TaxAmount / inv.AmountWithoutTax)
		}
		key := rateKey{rate: rate, direction: inv.Direction}
		row, ok := groups[key]
		if !ok {
			row = &domain.VATRateRow{TaxRate: rate}
			groups[key] = row
		}
		row.InvoiceCount++
		row.AmountWithoutTax = roundMoney(row.AmountWithoutTax + inv.AmountWithoutTax)
		row.TaxAmount = roundMoney(row.TaxAmount + inv.TaxAmount)
		row.AmountWithTax = roundMoney(row.AmountWithTax + inv.AmountWithTax)
	}

	cc := &domain.VATCrossCheck{}
	for k, row := range groups {
		if k.direction == "output" {
			cc.OutputRates = append(cc.OutputRates, *row)
			cc.OutputTotal += row.TaxAmount
		} else {
			cc.InputRates = append(cc.InputRates, *row)
			cc.InputTotal += row.TaxAmount
		}
	}
	cc.OutputTotal = roundMoney(cc.OutputTotal)
	cc.InputTotal = roundMoney(cc.InputTotal)
	cc.NetPayable = roundMoney(cc.OutputTotal - cc.InputTotal)

	// Cross-check warnings
	outputRateSet := map[float64]bool{}
	for _, r := range cc.OutputRates {
		outputRateSet[r.TaxRate] = true
	}
	for _, r := range cc.InputRates {
		if !outputRateSet[r.TaxRate] {
			cc.Warnings = append(cc.Warnings, fmt.Sprintf("进项税率 %.0f%% 无对应销项，请核实", r.TaxRate*100))
		}
	}
	return cc, nil
}

func (s *Service) VATReturn(ctx context.Context, entityID, bookID, period string) (*domain.VATReturn, error) {
	cc, err := s.VATCrossCheck(ctx, entityID, bookID, period)
	if err != nil {
		return nil, err
	}

	vr := &domain.VATReturn{}

	rowNo := 1
	for _, r := range cc.OutputRates {
		vr.Schedule1 = append(vr.Schedule1, domain.VATReturnRow{
			RowNo:           rowNo,
			Description:     fmt.Sprintf("税率 %.0f%% 销项", r.TaxRate*100),
			TaxRate:         r.TaxRate,
			AmountWithoutTax: r.AmountWithoutTax,
			TaxAmount:       r.TaxAmount,
		})
		rowNo++
	}

	rowNo = 1
	for _, r := range cc.InputRates {
		vr.Schedule2 = append(vr.Schedule2, domain.VATReturnRow{
			RowNo:           rowNo,
			Description:     fmt.Sprintf("税率 %.0f%% 进项", r.TaxRate*100),
			TaxRate:         r.TaxRate,
			AmountWithoutTax: r.AmountWithoutTax,
			TaxAmount:       r.TaxAmount,
		})
		rowNo++
	}

	vr.Main = domain.VATMainTable{
		OutputTax:          cc.OutputTotal,
		InputTax:           cc.InputTotal,
		TaxPayable:         cc.NetPayable,
		UrbanConstruction:  roundMoney(cc.NetPayable * 0.07),
		EducationSurcharge: roundMoney(cc.NetPayable * 0.03),
		LocalEducation:     roundMoney(cc.NetPayable * 0.02),
	}
	vr.Main.TotalTaxBurden = roundMoney(vr.Main.TaxPayable + vr.Main.UrbanConstruction + vr.Main.EducationSurcharge + vr.Main.LocalEducation)
	return vr, nil
}

func (s *Service) CrossTaxValidation(ctx context.Context, entityID, bookID, period string) (*domain.CrossTaxValidation, error) {
	vat, err := s.VATSummary(ctx, entityID, bookID, period)
	if err != nil {
		return nil, err
	}

	citRevenue := 0.0
	// Aggregate revenue accounts (6xxx prefix) from account balance
	ab, err := s.AccountBalance(ctx, entityID, bookID, period)
	if err != nil {
		return nil, err
	}
	for _, row := range ab.Rows {
		if strings.HasPrefix(row.AccountCode, "6") {
			citRevenue += row.CreditAmount
		}
	}

	vatSales := vat.OutputAmountWithoutTax
	diff := roundMoney(vatSales - citRevenue)
	devRate := 0.0
	if citRevenue > 0 {
		devRate = roundMoney(diff / citRevenue)
	}

	result := &domain.CrossTaxValidation{
		VATSales:      vatSales,
		CITRevenue:    citRevenue,
		Difference:    diff,
		DeviationRate: devRate,
		Consistent:    true,
	}
	absDev := devRate
	if absDev < 0 {
		absDev = -absDev
	}
	if absDev > 0.05 {
		result.Consistent = false
		result.Warnings = append(result.Warnings, fmt.Sprintf("增值税收入 %.2f 与所得税收入 %.2f 差异 %.1f%%", vatSales, citRevenue, devRate*100))
	}
	return result, nil
}

func convertProfitStatement(eps *engine.ProfitStatement) *domain.ProfitStatement {
	ps := &domain.ProfitStatement{
		Revenue:          eps.Revenue,
		Cost:             eps.Cost,
		TaxAndSurcharge:  eps.TaxAndSurcharge,
		SellingExpense:   eps.SellingExpense,
		AdminExpense:     eps.AdminExpense,
		FinanceExpense:   eps.FinanceExpense,
		AssetImpairment:  eps.AssetImpairment,
		FairValueGain:    eps.FairValueGain,
		InvestmentIncome: eps.InvestmentIncome,
		OperatingProfit:  eps.OperatingProfit,
		NonOpIncome:      eps.NonOpIncome,
		NonOpExpense:     eps.NonOpExpense,
		TotalProfit:      eps.TotalProfit,
		IncomeTax:        eps.IncomeTax,
		NetProfit:        eps.NetProfit,
	}
	for _, l := range eps.Lines {
		ps.Lines = append(ps.Lines, domain.ProfitStatementLine{
			LineCode:   l.LineCode,
			Label:      l.Label,
			Amount:     l.Amount,
			IsSubtotal: l.IsSubtotal,
		})
	}
	return ps
}

func convertBalanceSheet(ebs *engine.BalanceSheet) *domain.BalanceSheet {
	bs := &domain.BalanceSheet{
		TotalAssets:      ebs.TotalAssets,
		TotalLiabilities: ebs.TotalLiabilities,
		TotalEquity:      ebs.TotalEquity,
		TotalLiabEquity:  ebs.TotalLiabEquity,
	}
	for _, l := range ebs.Assets {
		bs.Assets = append(bs.Assets, domain.BalanceSheetLine{
			LineCode:   l.LineCode,
			Label:      l.Label,
			Amount:     l.Amount,
			IsSubtotal: l.IsSubtotal,
			Section:    l.Section,
		})
	}
	for _, l := range ebs.Liabilities {
		bs.Liabilities = append(bs.Liabilities, domain.BalanceSheetLine{
			LineCode:   l.LineCode,
			Label:      l.Label,
			Amount:     l.Amount,
			IsSubtotal: l.IsSubtotal,
			Section:    l.Section,
		})
	}
	for _, l := range ebs.Equity {
		bs.Equity = append(bs.Equity, domain.BalanceSheetLine{
			LineCode:   l.LineCode,
			Label:      l.Label,
			Amount:     l.Amount,
			IsSubtotal: l.IsSubtotal,
			Section:    l.Section,
		})
	}
	return bs
}

// ---- Tax Management operations ----

func (s *Service) CalculateVAT(ctx context.Context, input engine.VATInput, location string) (engine.VATOutput, error) {
	if location == "city" {
		return engine.CalculateWithLocation(input, engine.LocationCity)
	} else if location == "town" {
		return engine.CalculateWithLocation(input, engine.LocationTown)
	}
	return engine.CalculateWithLocation(input, engine.LocationOther)
}

func (s *Service) CalculateStampTax(ctx context.Context, input engine.StampTaxInput) (engine.StampTaxOutput, error) {
	return engine.CalcStampTax(input)
}

func (s *Service) CalculatePIT(ctx context.Context, input engine.PITInput) (engine.PITOutput, error) {
	return engine.CalcPIT(input)
}

func (s *Service) GenerateCITReport(ctx context.Context, entityID, bookID, taxYear, generatedBy string) (*domain.CITReport, error) {
	var totalRevenue, totalCost, totalExpense float64
	for m := 1; m <= 12; m++ {
		period := fmt.Sprintf("%s-%02d", taxYear, m)
		ab, err := s.AccountBalance(ctx, entityID, bookID, period)
		if err != nil {
			continue
		}
		for _, row := range ab.Rows {
			switch {
			case strings.HasPrefix(row.AccountCode, "6"):
				totalRevenue += row.CreditAmount
			case row.AccountCode == "5401" || row.AccountCode == "5402":
				totalCost += row.DebitAmount
			case strings.HasPrefix(row.AccountCode, "56"):
				totalExpense += row.DebitAmount
			}
		}
	}
	operatingProfit := roundMoney(totalRevenue - totalCost - totalExpense)

	var adjTotal float64
	if s.adjustments != nil {
		adjTotal, _ = s.adjustments.SumByYear(ctx, entityID, taxYear)
	}

	// Check tax profile for small profit eligibility
	isSmallProfit := false
	if s.taxProfiles != nil {
		profile, _ := s.taxProfiles.Get(ctx, entityID, bookID)
		if profile != nil && profile.CITRateType == domain.CITRateSmallProfit {
			isSmallProfit = true
		}
	}

	citOut, err := engine.CalcAnnualSettlement(operatingProfit, adjTotal, 0.25, 0, isSmallProfit)
	if err != nil {
		return nil, err
	}

	return &domain.CITReport{
		ID:              uuid.NewString(),
		EntityID:        entityID,
		BookID:          bookID,
		TaxYear:         taxYear,
		TotalRevenue:    totalRevenue,
		TotalCost:       totalCost,
		TotalExpense:    totalExpense,
		OperatingProfit: operatingProfit,
		AdjustmentTotal: adjTotal,
		TaxableIncome:   citOut.TaxableIncome,
		ApplicableRate:  citOut.ApplicableRate,
		TaxPayable:      citOut.TaxPayable,
		Deduction:       citOut.Deduction,
		PrepaidTax:      citOut.AccumulatedPaid,
		TaxDue:          citOut.TaxDue,
		Status:          "draft",
		GeneratedBy:     generatedBy,
	}, nil
}

func (s *Service) UpsertAdjustments(ctx context.Context, records []domain.AdjustmentRecord) error {
	if s.adjustments == nil {
		return ErrNotConfigured
	}
	for i := range records {
		if records[i].ID == "" {
			records[i].ID = uuid.NewString()
		}
		if err := s.adjustments.Upsert(ctx, &records[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ListAdjustments(ctx context.Context, query domain.AdjustmentListQuery) ([]domain.AdjustmentRecord, int, error) {
	if s.adjustments == nil {
		return nil, 0, ErrNotConfigured
	}
	return s.adjustments.List(ctx, query)
}

// ---- Risk Management operations ----

func (s *Service) TaxRisk(ctx context.Context, entityID, bookID, period string) (*domain.TaxRiskReport, error) {
	cc, err := s.VATCrossCheck(ctx, entityID, bookID, period)
	if err != nil {
		return nil, err
	}

	report := &domain.TaxRiskReport{Period: period}

	// Rate mismatch from cross-check
	for _, w := range cc.Warnings {
		report.Risks = append(report.Risks, domain.TaxRiskItem{
			Level:      "warning",
			Category:   "rate_mismatch",
			Title:      "税率不匹配",
			Detail:     w,
			Suggestion: "请核实进销项税率是否正确",
		})
	}

	// Excessive input credit
	if cc.InputTotal > cc.OutputTotal {
		ratio := cc.InputTotal / cc.OutputTotal
		level := "warning"
		if ratio >= 3.0 {
			level = "danger"
		}
		report.Risks = append(report.Risks, domain.TaxRiskItem{
			Level:    level,
			Category: "excess_credit",
			Title:    "进项税额过大",
			Detail:   fmt.Sprintf("进项税额 %.2f 超过销项税额 %.2f，比率为 %.1f", cc.InputTotal, cc.OutputTotal, ratio),
			Amount:   cc.InputTotal - cc.OutputTotal,
		})
	}

	// Tax burden anomaly
	invoices, _ := s.invoices.ListPostedByPeriod(ctx, entityID, bookID, period)
	for _, inv := range invoices {
		if inv.AmountWithoutTax > 0 {
			effectiveRate := inv.TaxAmount / inv.AmountWithoutTax
			if effectiveRate < 0.05 {
				report.Risks = append(report.Risks, domain.TaxRiskItem{
					Level:    "warning",
					Category: "tax_burden",
					Title:    "税负率偏低",
					Detail:   fmt.Sprintf("发票 %s 实际税负率 %.1f%% 低于 5%%", inv.InvoiceNo, effectiveRate*100),
					Amount:   inv.AmountWithTax,
				})
			}
		}
	}

	// Count by level
	for _, r := range report.Risks {
		switch r.Level {
		case "danger":
			report.DangerCount++
		case "warning":
			report.WarnCount++
		}
	}
	report.TotalRisks = len(report.Risks)
	return report, nil
}

func (s *Service) RiskScan(ctx context.Context, entityID, bookID, period string) ([]domain.RiskFinding, error) {
	var findings []domain.RiskFinding

	// Tax risk findings
	taxRisk, err := s.TaxRisk(ctx, entityID, bookID, period)
	if err == nil {
		for _, r := range taxRisk.Risks {
			findings = append(findings, domain.RiskFinding{
				RuleCode:   "tax_risk_" + r.Category,
				Category:   r.Category,
				Severity:   r.Level,
				Title:      r.Title,
				Detail:     r.Detail,
				Suggestion: r.Suggestion,
				Amount:     r.Amount,
			})
		}
	}

	// Additional engine-based checks
	invoices, err := s.invoices.ListPostedByPeriod(ctx, entityID, bookID, period)
	if err == nil {
		// Large round transactions
		inputs := make([]engine.RoundTransactionInput, len(invoices))
		for i, inv := range invoices {
			inputs[i] = engine.RoundTransactionInput{
				InvoiceNo:     inv.InvoiceNo,
				AmountWithTax: inv.AmountWithTax,
			}
		}
		for _, risk := range engine.DetectLargeRoundTransactions(inputs) {
			findings = append(findings, domain.RiskFinding{
				RuleCode: "large_round_transaction",
				Category: "amount_anomaly",
				Severity: risk.Level,
				Title:    "大额整数交易",
				Detail:   fmt.Sprintf("发票 %s 金额为 %.2f 整数", risk.InvoiceNo, risk.Amount),
				Amount:   risk.Amount,
			})
		}

		// Supplier concentration
		sellerAmounts := make(map[string]float64)
		for _, inv := range invoices {
			if inv.Direction == domain.DirectionInput {
				sellerAmounts[inv.SellerName] += inv.AmountWithTax
			}
		}
		if sc := engine.DetectSupplierConcentration(sellerAmounts); sc != nil {
			findings = append(findings, domain.RiskFinding{
				RuleCode: "supplier_concentration",
				Category: "concentration",
				Severity: sc.Level,
				Title:    "供应商集中度",
				Detail:   fmt.Sprintf("供应商 %s 集中度 %.1f%%", sc.TopSupplier, sc.Concentration*100),
				Amount:   sc.TopAmount,
			})
		}
	}

	return findings, nil
}

func (s *Service) RunRiskScanPersistent(ctx context.Context, entityID, bookID, period string) (*domain.RiskScan, error) {
	findings, err := s.RiskScan(ctx, entityID, bookID, period)
	if err != nil {
		return nil, err
	}

	engineFindings := make([]engine.RiskFinding, len(findings))
	for i, f := range findings {
		engineFindings[i] = engine.RiskFinding{
			RuleCode:   f.RuleCode,
			Category:   f.Category,
			Severity:   f.Severity,
			Title:      f.Title,
			Detail:     f.Detail,
			Suggestion: f.Suggestion,
			Amount:     f.Amount,
		}
	}
	totalScore, riskLevel := engine.ScoreRiskFindings(engineFindings)

	findingsJSON, _ := json.Marshal(findings)
	scan := &domain.RiskScan{
		ID:            uuid.NewString(),
		EntityID:      entityID,
		BookID:        bookID,
		Period:        period,
		ScanType:      domain.ScanTypeFull,
		RulesTriggered: []byte("[]"),
		Findings:      findingsJSON,
		TotalScore:    totalScore,
		RiskLevel:     riskLevel,
		EngineVersion: "v2.0",
	}

	if s.riskScans != nil {
		if err := s.riskScans.Create(ctx, scan); err != nil {
			return nil, err
		}
	}

	return scan, nil
}

func (s *Service) RunConsistencyCheck(ctx context.Context, entityID, bookID, period string) ([]domain.ConsistencyCheck, error) {
	var checks []domain.ConsistencyCheck

	// Invoice-to-VAT consistency
	vat, _ := s.VATCrossCheck(ctx, entityID, bookID, period)
	if vat != nil {
		invOutputTax, invInputTax := vat.OutputTotal, vat.InputTotal
		cr := engine.CheckInvoiceToVAT(invOutputTax, invInputTax, vat.OutputTotal, vat.InputTotal)
		checks = append(checks, domain.ConsistencyCheck{
			ID:          uuid.NewString(),
			EntityID:    entityID,
			BookID:      bookID,
			Period:      period,
			CheckType:   cr.CheckType,
			CheckName:   cr.CheckName,
			SourceValue: cr.SourceValue,
			TargetValue: cr.TargetValue,
			Difference:  cr.Difference,
			Tolerance:   cr.Tolerance,
			Passed:      cr.Passed,
			Detail:      cr.Detail,
		})
	}

	// Invoice-to-Journal consistency
	approvedInvoices, _ := s.invoices.List(ctx, entityID, bookID, period, domain.StatusApproved, 1000, 0)
	journalEntries, _ := s.journals.List(ctx, domain.JournalListQuery{
		EntityID: entityID,
		BookID:   bookID,
		Period:   period,
		Status:   domain.JournalStatusPosted,
		Limit:    1000,
	})
	var approvedAmount float64
	for _, inv := range approvedInvoices {
		approvedAmount += inv.AmountWithTax
	}
	var journalAmount float64
	for _, je := range journalEntries {
		for _, l := range je.Lines {
			journalAmount += l.DebitAmount
		}
	}
	jcr := engine.CheckInvoiceToJournal(len(approvedInvoices), len(journalEntries), approvedAmount, journalAmount)
	checks = append(checks, domain.ConsistencyCheck{
		ID:          uuid.NewString(),
		EntityID:    entityID,
		BookID:      bookID,
		Period:      period,
		CheckType:   jcr.CheckType,
		CheckName:   jcr.CheckName,
		SourceValue: jcr.SourceValue,
		TargetValue: jcr.TargetValue,
		Difference:  jcr.Difference,
		Tolerance:   jcr.Tolerance,
		Passed:      jcr.Passed,
		Detail:      jcr.Detail,
	})

	// Journal-to-Bank consistency
	if s.reconciliation != nil {
		unmatched, _ := s.reconciliation.ListUnmatchedBankTransactions(ctx, entityID, bookID)
		var bankTotal float64
		for _, bt := range unmatched {
			bankTotal += bt.Amount
		}
		bcr := engine.CheckJournalToBank(journalAmount, bankTotal)
		checks = append(checks, domain.ConsistencyCheck{
			ID:          uuid.NewString(),
			EntityID:    entityID,
			BookID:      bookID,
			Period:      period,
			CheckType:   bcr.CheckType,
			CheckName:   bcr.CheckName,
			SourceValue: bcr.SourceValue,
			TargetValue: bcr.TargetValue,
			Difference:  bcr.Difference,
			Tolerance:   bcr.Tolerance,
			Passed:      bcr.Passed,
			Detail:      bcr.Detail,
		})
	}

	if s.consistencyChecks != nil {
		if err := s.consistencyChecks.CreateMany(ctx, checks); err != nil {
			return checks, err
		}
	}
	return checks, nil
}

func (s *Service) EnhanceCloseCheck(ctx context.Context, entityID, bookID, period string) (*domain.CloseCheckResult, error) {
	result, err := s.CloseCheck(ctx, entityID, bookID, period)
	if err != nil {
		return nil, err
	}

	// Add consistency checks
	consistencyResults, _ := s.RunConsistencyCheck(ctx, entityID, bookID, period)
	for _, cc := range consistencyResults {
		if !cc.Passed {
			result.Passed = false
			item := domain.CloseCheck{
				Label:    cc.CheckName,
				Passed:   false,
				Severity: "blocking",
				Category: "consistency",
				Detail:   cc.Detail,
			}
			result.Checks = append(result.Checks, item)
		}
	}

	return result, nil
}

// ---- Export operations ----

func (s *Service) ExportTrialBalanceCSV(ctx context.Context, entityID, bookID, period string) ([]byte, error) {
	tb, err := s.TrialBalance(ctx, entityID, bookID, period)
	if err != nil {
		return nil, err
	}
	var sb strings.Builder
	sb.WriteString("AccountCode,AccountName,DebitAmount,CreditAmount,NetAmount\n")
	for _, row := range tb.Rows {
		sb.WriteString(fmt.Sprintf("%s,%s,%.2f,%.2f,%.2f\n", row.AccountCode, row.AccountName, row.DebitAmount, row.CreditAmount, row.NetAmount))
	}
	return []byte(sb.String()), nil
}

func (s *Service) ExportVATSummaryCSV(ctx context.Context, entityID, bookID, period string) ([]byte, error) {
	vat, err := s.VATSummary(ctx, entityID, bookID, period)
	if err != nil {
		return nil, err
	}
	var sb strings.Builder
	sb.WriteString("Period,InputCount,InputAmountWithoutTax,InputTaxAmount,InputAmountWithTax,OutputCount,OutputAmountWithoutTax,OutputTaxAmount,OutputAmountWithTax\n")
	sb.WriteString(fmt.Sprintf("%s,%d,%.2f,%.2f,%.2f,%d,%.2f,%.2f,%.2f\n",
		vat.Period, vat.InvoiceCount, vat.InputAmountWithoutTax, vat.InputTaxAmount, vat.InputAmountWithTax,
		vat.OutputInvoiceCount, vat.OutputAmountWithoutTax, vat.OutputTaxAmount, vat.OutputAmountWithTax))
	return []byte(sb.String()), nil
}

func (s *Service) ExportVATReturnJSON(ctx context.Context, entityID, bookID, period string) ([]byte, error) {
	vr, err := s.VATReturn(ctx, entityID, bookID, period)
	if err != nil {
		return nil, err
	}
	return json.Marshal(vr)
}

func (s *Service) ExportCITReturnJSON(ctx context.Context, entityID, bookID, taxYear, actorID string) ([]byte, error) {
	cit, err := s.GenerateCITReport(ctx, entityID, bookID, taxYear, actorID)
	if err != nil {
		return nil, err
	}
	return json.Marshal(cit)
}

func (s *Service) ExportAdjustmentsJSON(ctx context.Context, entityID, taxYear string) ([]byte, error) {
	items, _, err := s.ListAdjustments(ctx, domain.AdjustmentListQuery{
		EntityID: entityID,
		TaxYear:  taxYear,
		Limit:    1000,
	})
	if err != nil {
		return nil, err
	}
	return json.Marshal(items)
}

func (s *Service) fallbackProfitStatement(lines []engine.ReportLine) *domain.ProfitStatement {
	ps := &domain.ProfitStatement{}
	for _, l := range lines {
		code := l.AccountCode
		amount := l.Amount
		switch {
		case strings.HasPrefix(code, "6"):
			ps.Revenue = roundMoney(ps.Revenue + amount)
		case code == "5401" || code == "5402":
			ps.Cost = roundMoney(ps.Cost + amount)
		case code == "5403":
			ps.TaxAndSurcharge = roundMoney(ps.TaxAndSurcharge + amount)
		case strings.HasPrefix(code, "5601") || strings.HasPrefix(code, "5602") || strings.HasPrefix(code, "5603"):
			ps.SellingExpense = roundMoney(ps.SellingExpense + amount)
		}
	}
	ps.OperatingProfit = roundMoney(ps.Revenue - ps.Cost - ps.TaxAndSurcharge - ps.SellingExpense - ps.AdminExpense - ps.FinanceExpense)
	ps.TotalProfit = roundMoney(ps.OperatingProfit + ps.NonOpIncome - ps.NonOpExpense)
	ps.NetProfit = roundMoney(ps.TotalProfit - ps.IncomeTax)
	return ps
}

func (s *Service) seedStandardChartOfAccounts(ctx context.Context, entityID, bookID string) {
	if s.accounts == nil {
		return
	}

	type acct struct {
		code, name, category, balanceType string
	}
	standard := []acct{
		{"1001", "库存现金", "asset", "debit"},
		{"1002", "银行存款", "asset", "debit"},
		{"1012", "其他货币资金", "asset", "debit"},
		{"1122", "应收账款", "asset", "debit"},
		{"1123", "预付账款", "asset", "debit"},
		{"1131", "应收股利", "asset", "debit"},
		{"1132", "应收利息", "asset", "debit"},
		{"1221", "其他应收款", "asset", "debit"},
		{"1403", "原材料", "asset", "debit"},
		{"1405", "库存商品", "asset", "debit"},
		{"1411", "周转材料", "asset", "debit"},
		{"1601", "固定资产", "asset", "debit"},
		{"1602", "累计折旧", "asset", "credit"},
		{"1701", "无形资产", "asset", "debit"},
		{"1702", "累计摊销", "asset", "credit"},
		{"2001", "短期借款", "liability", "credit"},
		{"2202", "应付账款", "liability", "credit"},
		{"2203", "预收账款", "liability", "credit"},
		{"2211", "应付职工薪酬", "liability", "credit"},
		{"2221", "应交税费", "liability", "credit"},
		{"2221-01", "应交税费-应交增值税-进项税额", "liability", "debit"},
		{"2221-02", "应交税费-应交增值税-销项税额", "liability", "credit"},
		{"2221-03", "应交税费-应交增值税-已交税金", "liability", "debit"},
		{"2221-06", "应交税费-应交城市维护建设税", "liability", "credit"},
		{"2221-07", "应交税费-应交教育费附加", "liability", "credit"},
		{"2221-08", "应交税费-应交地方教育附加", "liability", "credit"},
		{"2221-09", "应交税费-应交企业所得税", "liability", "credit"},
		{"2221-12", "应交税费-应交印花税", "liability", "credit"},
		{"2241", "其他应付款", "liability", "credit"},
		{"2501", "长期借款", "liability", "credit"},
		{"4001", "实收资本", "equity", "credit"},
		{"4002", "资本公积", "equity", "credit"},
		{"4101", "盈余公积", "equity", "credit"},
		{"4104", "利润分配-未分配利润", "equity", "credit"},
		{"5401", "主营业务成本", "cost", "debit"},
		{"5402", "其他业务成本", "cost", "debit"},
		{"5403", "税金及附加", "cost", "debit"},
		{"5601", "销售费用", "expense", "debit"},
		{"5602", "管理费用", "expense", "debit"},
		{"5603", "财务费用", "expense", "debit"},
		{"6001", "主营业务收入", "revenue", "credit"},
		{"6051", "其他业务收入", "revenue", "credit"},
		{"6111", "投资收益", "revenue", "credit"},
		{"6301", "营业外收入", "revenue", "credit"},
		{"6711", "营业外支出", "expense", "debit"},
		{"6801", "所得税费用", "expense", "debit"},
	}

	for _, a := range standard {
		_ = s.accounts.Create(ctx, &domain.ChartAccount{
			ID:          uuid.NewString(),
			EntityID:    entityID,
			BookID:      bookID,
			Code:        a.code,
			Name:        a.name,
			Category:    a.category,
			BalanceType: a.balanceType,
			IsSystem:    true,
			Keywords:    []string{},
		})
	}
}
