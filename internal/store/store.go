package store

import (
	"context"

	"finance.chao.run/v2/internal/domain"

	"github.com/jackc/pgx/v5"
)

type BookStore interface {
	Create(ctx context.Context, book *domain.AccountingBook) error
	GetByID(ctx context.Context, entityID, bookID string) (*domain.AccountingBook, error)
	GetDefault(ctx context.Context, entityID string) (*domain.AccountingBook, error)
	List(ctx context.Context, entityID string) ([]domain.AccountingBook, error)
	Update(ctx context.Context, book *domain.AccountingBook) error
	SetDefault(ctx context.Context, entityID, bookID string) error
}

type PeriodStore interface {
	GetOrCreate(ctx context.Context, p *domain.AccountingPeriod) (*domain.AccountingPeriod, error)
	Get(ctx context.Context, entityID, bookID, period string) (*domain.AccountingPeriod, error)
	ListByBook(ctx context.Context, entityID, bookID string) ([]domain.AccountingPeriod, error)
	IsClosed(ctx context.Context, entityID, bookID, period string) (bool, error)
	UpdateStatus(ctx context.Context, entityID, bookID, period, fromStatus, toStatus string, closedBy *string) (*domain.AccountingPeriod, error)
}

type InvoiceStore interface {
	Create(ctx context.Context, inv *domain.Invoice) error
	Update(ctx context.Context, inv *domain.Invoice) error
	Get(ctx context.Context, entityID, invoiceID string) (*domain.Invoice, error)
	List(ctx context.Context, entityID, bookID, period, status string, limit, offset int) ([]domain.Invoice, error)
	FindByInvoiceNo(ctx context.Context, entityID, bookID, invoiceNo string) (*domain.Invoice, error)
	FindByDigitalInvoiceNo(ctx context.Context, entityID, bookID, digitalNo string) (*domain.Invoice, error)
	FindByOriginalInvoiceID(ctx context.Context, entityID, originalInvoiceID string) ([]domain.Invoice, error)
	Approve(ctx context.Context, entityID, invoiceID string) (*domain.Invoice, error)
	ApproveTx(ctx context.Context, tx pgx.Tx, entityID, invoiceID string) (*domain.Invoice, error)
	Reject(ctx context.Context, entityID, invoiceID string) (*domain.Invoice, error)
	MarkPosted(ctx context.Context, entityID, invoiceID string) (*domain.Invoice, error)
	MarkPostedTx(ctx context.Context, tx pgx.Tx, entityID, invoiceID string) (*domain.Invoice, error)
	UpdateVerificationStatus(ctx context.Context, entityID, invoiceID, status string) error
	UpdateUsageStatus(ctx context.Context, entityID, invoiceID, status string) error
	UpdateRedLetterStatus(ctx context.Context, entityID, invoiceID, status string) error
	UpdateDeductionStatus(ctx context.Context, entityID, invoiceID, status string) error
	ListPostedByPeriod(ctx context.Context, entityID, bookID, period string) ([]domain.Invoice, error)
}

type InvoiceLineStore interface {
	CreateMany(ctx context.Context, lines []domain.InvoiceLine) error
	CreateManyTx(ctx context.Context, tx pgx.Tx, lines []domain.InvoiceLine) error
	UpsertMany(ctx context.Context, lines []domain.InvoiceLine) error
	DeleteByInvoice(ctx context.Context, entityID, invoiceID string) error
	ListByInvoice(ctx context.Context, entityID, invoiceID string) ([]domain.InvoiceLine, error)
}

type AccountStore interface {
	Create(ctx context.Context, a *domain.ChartAccount) error
	GetByCode(ctx context.Context, entityID, bookID, code string) (*domain.ChartAccount, error)
	List(ctx context.Context, query domain.AccountListQuery) ([]domain.ChartAccount, int, error)
	Update(ctx context.Context, a *domain.ChartAccount) error
	Delete(ctx context.Context, entityID, accountID string) error
}

type JournalStore interface {
	Create(ctx context.Context, entry *domain.JournalEntry) error
	CreateTx(ctx context.Context, tx pgx.Tx, entry *domain.JournalEntry) error
	Get(ctx context.Context, entityID, journalID string) (*domain.JournalEntry, error)
	List(ctx context.Context, query domain.JournalListQuery) ([]domain.JournalEntry, error)
	Post(ctx context.Context, entityID, journalID string) (*domain.JournalEntry, error)
	PostTx(ctx context.Context, tx pgx.Tx, entityID, journalID string) (*domain.JournalEntry, error)
	Void(ctx context.Context, entityID, journalID string) (*domain.JournalEntry, error)
	VoidTx(ctx context.Context, tx pgx.Tx, entityID, journalID string) (*domain.JournalEntry, error)
	ListPostedLines(ctx context.Context, entityID, bookID, period string) ([]domain.JournalLine, error)
	NextVoucherNo(ctx context.Context, entityID, bookID, period string) (string, error)
}

type AuditStore interface {
	Append(ctx context.Context, e *domain.AuditEntry) error
	List(ctx context.Context, query domain.AuditListQuery) ([]domain.AuditEntry, error)
}

type ReportMappingStore interface {
	ListByReportType(ctx context.Context, reportType, accountingStandard string) ([]domain.ReportMapping, error)
	ListAll(ctx context.Context) ([]domain.ReportMapping, error)
}

type AdjustmentStore interface {
	Upsert(ctx context.Context, rec *domain.AdjustmentRecord) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, query domain.AdjustmentListQuery) ([]domain.AdjustmentRecord, int, error)
	DeleteByCategory(ctx context.Context, entityID, taxYear, category string) error
	SumByYear(ctx context.Context, entityID, taxYear string) (float64, error)
}

type TaxProfileStore interface {
	Upsert(ctx context.Context, p *domain.TaxProfile) error
	Get(ctx context.Context, entityID, bookID string) (*domain.TaxProfile, error)
}

type TaxReturnStore interface {
	Upsert(ctx context.Context, r *domain.TaxReturn) error
	Get(ctx context.Context, entityID, bookID, taxYear, taxPeriod, returnType string) (*domain.TaxReturn, error)
}

type RiskScanStore interface {
	Create(ctx context.Context, scan *domain.RiskScan) error
	ListByPeriod(ctx context.Context, entityID, bookID, period string) ([]domain.RiskScan, error)
	GetLatest(ctx context.Context, entityID, bookID, period string) (*domain.RiskScan, error)
}

type ConsistencyCheckStore interface {
	CreateMany(ctx context.Context, checks []domain.ConsistencyCheck) error
	ListByPeriod(ctx context.Context, entityID, bookID, period string) ([]domain.ConsistencyCheck, error)
}

type ReconciliationStore interface {
	UpsertLogistics(ctx context.Context, lr *domain.LogisticsRecord) error
	GetLogisticsByInvoice(ctx context.Context, entityID, invoiceID string) (*domain.LogisticsRecord, error)
	DeleteLogistics(ctx context.Context, entityID, invoiceID string) error
	UpsertBankTransaction(ctx context.Context, bt *domain.BankTransaction) error
	GetBankTransaction(ctx context.Context, entityID, bankTxID string) (*domain.BankTransaction, error)
	MatchBankToInvoice(ctx context.Context, entityID, bankTxID, invoiceID string, confidence float64) error
	UnmatchBankFromInvoice(ctx context.Context, entityID, bankTxID string) error
	ListUnmatchedBankTransactions(ctx context.Context, entityID, bookID string) ([]domain.BankTransaction, error)
	ThreeWayMatch(ctx context.Context, entityID, bookID, period string) (*domain.ThreeWaySummary, error)
}

type IdempotencyStore interface {
	Get(ctx context.Context, entityID, capabilityID, idempotencyKey string) (*IdempotencyRecord, error)
	Save(ctx context.Context, r *IdempotencyRecord) error
}

type IdempotencyRecord struct {
	EntityID       string
	CapabilityID   string
	IdempotencyKey string
	InputHash      string
	Result         []byte
	Status         string
	CreatedAt      string
}
