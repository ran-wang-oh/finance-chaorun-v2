package postgres

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"finance.chao.run/v2/internal/domain"
	"finance.chao.run/v2/internal/store"

	"github.com/google/uuid"
)

func testDSN() string {
	if dsn := os.Getenv("DATABASE_DSN"); dsn != "" {
		return dsn
	}
	return "postgres://chaorun:chaorun_dev@localhost:5432/chaorun_finance?sslmode=disable"
}

func connectTestDB(t *testing.T) *DB {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, err := Connect(context.Background(), testDSN())
	if err != nil {
		t.Skipf("cannot connect to test database: %v (set DATABASE_DSN or run docker compose up)", err)
	}
	if err := db.RunMigrations(context.Background()); err != nil {
		t.Fatalf("migrations failed: %v", err)
	}
	return db
}

func entityA() string { return uuid.NewString() }
func entityB() string { return uuid.NewString() }

// ——— Book ownership & entity scoping ———

func TestBookStore_EntityScopedGetByID(t *testing.T) {
	db := connectTestDB(t)
	defer db.Close()
	books := NewBookStore(db)

	e1, e2 := entityA(), entityB()
	b1 := &domain.AccountingBook{
		ID: uuid.NewString(), EntityID: e1, Code: uuid.NewString()[:8],
		Name: "Entity 1 Book", AccountingStandard: "small_business_gaap_cn",
		BaseCurrency: "CNY", StartPeriod: "2025-01",
	}
	b2 := &domain.AccountingBook{
		ID: uuid.NewString(), EntityID: e2, Code: uuid.NewString()[:8],
		Name: "Entity 2 Book", AccountingStandard: "small_business_gaap_cn",
		BaseCurrency: "CNY", StartPeriod: "2025-01",
	}
	if err := books.Create(context.Background(), b1); err != nil {
		t.Fatalf("create b1: %v", err)
	}
	if err := books.Create(context.Background(), b2); err != nil {
		t.Fatalf("create b2: %v", err)
	}

	// e1 can get its own book
	got, err := books.GetByID(context.Background(), e1, b1.ID)
	if err != nil {
		t.Fatalf("GetByID for own entity: %v", err)
	}
	if got.ID != b1.ID {
		t.Errorf("expected book %s, got %s", b1.ID, got.ID)
	}

	// e1 cannot get e2's book by ID (returns nil, nil for no rows)
	got, err = books.GetByID(context.Background(), e1, b2.ID)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if got != nil {
		t.Errorf("expected nil book for cross-entity access, got %+v", got)
	}
}

func TestBookStore_EntityScopedList(t *testing.T) {
	db := connectTestDB(t)
	defer db.Close()
	books := NewBookStore(db)

	e1 := entityA()
	b1 := &domain.AccountingBook{
		ID: uuid.NewString(), EntityID: e1, Code: uuid.NewString()[:8],
		Name: "List Test Book", AccountingStandard: "small_business_gaap_cn",
		BaseCurrency: "CNY", StartPeriod: "2025-01",
	}
	if err := books.Create(context.Background(), b1); err != nil {
		t.Fatalf("create book: %v", err)
	}

	list, err := books.List(context.Background(), e1)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, b := range list {
		if b.ID == b1.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("created book not found in entity list")
	}

	// Wrong entity should not see it
	list, err = books.List(context.Background(), entityB())
	if err != nil {
		t.Fatalf("List for unrelated entity: %v", err)
	}
	for _, b := range list {
		if b.ID == b1.ID {
			t.Error("book from entity A leaked into entity B's list")
		}
	}
}

func TestInvoiceStore_EntityScopedGet(t *testing.T) {
	db := connectTestDB(t)
	defer db.Close()
	invoices := NewInvoiceStore(db)

	e1, e2 := entityA(), entityB()
	inv1 := &domain.Invoice{
		ID: uuid.NewString(), EntityID: e1, BookID: "book-1",
		InvoiceNo: uuid.NewString()[:12], Direction: domain.DirectionInput,
		IssueDate: "2026-03-15", Status: domain.StatusPendingReview,
		AmountWithoutTax: 100, TaxAmount: 13, AmountWithTax: 113,
	}
	if err := invoices.Create(context.Background(), inv1); err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	// Own entity can fetch
	got, err := invoices.Get(context.Background(), e1, inv1.ID)
	if err != nil {
		t.Fatalf("Get own invoice: %v", err)
	}
	if got.ID != inv1.ID {
		t.Errorf("expected %s, got %s", inv1.ID, got.ID)
	}

	// Cross-entity cannot fetch
	got, err = invoices.Get(context.Background(), e2, inv1.ID)
	if err == nil && got != nil {
		t.Error("cross-entity Get should return no rows")
	}
}

// ——— Idempotency ———

func TestIdempotencyStore_SaveAndReplay(t *testing.T) {
	db := connectTestDB(t)
	defer db.Close()
	idem := NewIdempotencyStore(db)

	e1 := entityA()
	rec := &store.IdempotencyRecord{
		EntityID: e1, CapabilityID: "finance.book.create",
		IdempotencyKey: "idem-test-1", InputHash: "abc123",
		Result: json.RawMessage(`{"id":"book-x"}`), Status: "completed",
	}
	if err := idem.Save(context.Background(), rec); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Replay fetches record
	got, err := idem.Get(context.Background(), e1, "finance.book.create", "idem-test-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected record, got nil")
	}
	if got.InputHash != "abc123" {
		t.Errorf("expected hash abc123, got %s", got.InputHash)
	}
	if got.Status != "completed" {
		t.Errorf("expected status completed, got %s", got.Status)
	}
}

func TestIdempotencyStore_ConflictDifferentInput(t *testing.T) {
	db := connectTestDB(t)
	defer db.Close()
	idem := NewIdempotencyStore(db)

	e1 := entityA()
	rec := &store.IdempotencyRecord{
		EntityID: e1, CapabilityID: "finance.invoice.create_draft",
		IdempotencyKey: "idem-conflict-test", InputHash: "original-hash",
		Result: json.RawMessage(`{"invoice_id":"inv-1"}`), Status: "completed",
	}
	if err := idem.Save(context.Background(), rec); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Fetch and compare hash
	got, err := idem.Get(context.Background(), e1, "finance.invoice.create_draft", "idem-conflict-test")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected record")
	}
	if got.InputHash == "different-hash" {
		t.Error("hash should not match different input")
	}
}

func TestIdempotencyStore_DifferentEntitiesIsolated(t *testing.T) {
	db := connectTestDB(t)
	defer db.Close()
	idem := NewIdempotencyStore(db)

	e1, e2 := entityA(), entityB()
	rec := &store.IdempotencyRecord{
		EntityID: e1, CapabilityID: "finance.book.create",
		IdempotencyKey: "same-key-across-entities", InputHash: "h1",
		Result: json.RawMessage(`{}`), Status: "completed",
	}
	if err := idem.Save(context.Background(), rec); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// e2 with same key should not find e1's record
	got, err := idem.Get(context.Background(), e2, "finance.book.create", "same-key-across-entities")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != nil {
		t.Error("entity B should not see entity A's idempotency record")
	}
}

// ——— Journal post ———

func TestJournalStore_CreateAndPost(t *testing.T) {
	db := connectTestDB(t)
	defer db.Close()
	journals := NewJournalStore(db)

	e1 := entityA()
	entry := &domain.JournalEntry{
		ID: uuid.NewString(), EntityID: e1, BookID: "book-1",
		Period: "2026-03", Status: domain.JournalStatusDraft,
		Summary: "test journal", EntryDate: "2026-03-15",
		VoucherWord: domain.VoucherWordJi, VoucherNo: "1",
		Lines: []domain.JournalLine{
			{ID: uuid.NewString(), EntityID: e1, JournalEntryID: "", AccountCode: "1001",
				AccountName: "Cash", Direction: domain.DirectionDebit, DebitAmount: 100, LineNo: 1},
			{ID: uuid.NewString(), EntityID: e1, JournalEntryID: "", AccountCode: "6001",
				AccountName: "Revenue", Direction: domain.DirectionCredit, CreditAmount: 100, LineNo: 2},
		},
	}

	if err := journals.Create(context.Background(), entry); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Fetch draft
	got, err := journals.Get(context.Background(), e1, entry.ID)
	if err != nil {
		t.Fatalf("Get draft: %v", err)
	}
	if got.Status != domain.JournalStatusDraft {
		t.Errorf("expected draft, got %s", got.Status)
	}

	// Post
	posted, err := journals.Post(context.Background(), e1, entry.ID)
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if posted.Status != domain.JournalStatusPosted {
		t.Errorf("expected posted, got %s", posted.Status)
	}
}

func TestJournalStore_PostedLinesScopedByEntity(t *testing.T) {
	db := connectTestDB(t)
	defer db.Close()
	journals := NewJournalStore(db)

	e1 := entityA()
	entry := &domain.JournalEntry{
		ID: uuid.NewString(), EntityID: e1, BookID: "book-1",
		Period: "2026-03", Status: domain.JournalStatusDraft,
		Summary: "entity scope test", EntryDate: "2026-03-15",
		VoucherWord: domain.VoucherWordJi, VoucherNo: "2",
		Lines: []domain.JournalLine{
			{ID: uuid.NewString(), EntityID: e1, JournalEntryID: "", AccountCode: "1001",
				AccountName: "Cash", Direction: domain.DirectionDebit, DebitAmount: 50, LineNo: 1},
			{ID: uuid.NewString(), EntityID: e1, JournalEntryID: "", AccountCode: "6001",
				AccountName: "Revenue", Direction: domain.DirectionCredit, CreditAmount: 50, LineNo: 2},
		},
	}

	if err := journals.Create(context.Background(), entry); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := journals.Post(context.Background(), e1, entry.ID); err != nil {
		t.Fatalf("Post: %v", err)
	}

	// Posted lines for correct entity
	lines, err := journals.ListPostedLines(context.Background(), e1, "book-1", "2026-03")
	if err != nil {
		t.Fatalf("ListPostedLines: %v", err)
	}
	if len(lines) != 2 {
		t.Errorf("expected 2 posted lines for e1, got %d", len(lines))
	}

	// Different entity should see no lines
	lines, err = journals.ListPostedLines(context.Background(), entityB(), "book-1", "2026-03")
	if err != nil {
		t.Fatalf("ListPostedLines for unrelated entity: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("entity B should see 0 posted lines, got %d", len(lines))
	}
}

// ——— Trial balance ———

func TestTrialBalance_Balanced(t *testing.T) {
	db := connectTestDB(t)
	defer db.Close()
	journals := NewJournalStore(db)

	e1 := entityA()
	entry := &domain.JournalEntry{
		ID: uuid.NewString(), EntityID: e1, BookID: "book-1",
		Period: "2026-03", Status: domain.JournalStatusDraft,
		Summary: "trial balance test", EntryDate: "2026-03-15",
		VoucherWord: domain.VoucherWordJi, VoucherNo: "3",
		Lines: []domain.JournalLine{
			{ID: uuid.NewString(), EntityID: e1, JournalEntryID: "", AccountCode: "1001",
				AccountName: "Cash", Direction: domain.DirectionDebit, DebitAmount: 200, LineNo: 1},
			{ID: uuid.NewString(), EntityID: e1, JournalEntryID: "", AccountCode: "6001",
				AccountName: "Revenue", Direction: domain.DirectionCredit, CreditAmount: 200, LineNo: 2},
		},
	}

	if err := journals.Create(context.Background(), entry); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := journals.Post(context.Background(), e1, entry.ID); err != nil {
		t.Fatalf("Post: %v", err)
	}

	lines, err := journals.ListPostedLines(context.Background(), e1, "book-1", "2026-03")
	if err != nil {
		t.Fatalf("ListPostedLines: %v", err)
	}

	var totalDebit, totalCredit float64
	for _, l := range lines {
		totalDebit += l.DebitAmount
		totalCredit += l.CreditAmount
	}
	if totalDebit != 200 || totalCredit != 200 {
		t.Errorf("trial balance mismatch: debit=%.2f credit=%.2f", totalDebit, totalCredit)
	}
}

// ——— Audit ———

func TestAuditStore_AppendAndList(t *testing.T) {
	db := connectTestDB(t)
	defer db.Close()
	audit := NewAuditStore(db)

	e1 := entityA()
	entry := &domain.AuditEntry{
		ID: uuid.NewString(), EntityID: e1, BookID: "book-1",
		CapabilityID: "finance.book.create", V2CapabilityID: "finance.book.create",
		TraceID: "trace-001", IdempotencyKey: "idem-audit-1",
		ActorType: "user", ActorID: "actor-1",
		Action: domain.AuditActionBookCreate, ObjectType: "book",
		ObjectID: "book-1", Outcome: domain.AuditOutcomeSuccess,
		Payload: []byte(`{"ref":"book-1","details":""}`),
	}
	if err := audit.Append(context.Background(), entry); err != nil {
		t.Fatalf("Append: %v", err)
	}

	// List with entity filter
	list, err := audit.List(context.Background(), domain.AuditListQuery{EntityID: e1, Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("expected at least 1 audit entry")
	}

	last := list[len(list)-1]
	if last.Action != domain.AuditActionBookCreate {
		t.Errorf("expected action book.create, got %s", last.Action)
	}
	if last.TraceID != "trace-001" {
		t.Errorf("expected trace_id trace-001, got %s", last.TraceID)
	}

	// Different entity should not see it
	list, err = audit.List(context.Background(), domain.AuditListQuery{EntityID: entityB(), Limit: 10})
	if err != nil {
		t.Fatalf("List for unrelated entity: %v", err)
	}
	for _, e := range list {
		if e.ID == entry.ID {
			t.Error("audit entry leaked across entity boundaries")
		}
	}
}

func TestAuditStore_FullMetadata(t *testing.T) {
	db := connectTestDB(t)
	defer db.Close()
	audit := NewAuditStore(db)

	e1 := entityA()
	entry := &domain.AuditEntry{
		ID: uuid.NewString(), EntityID: e1, BookID: "book-1",
		CapabilityID: "finance.invoice.approve",
		V2CapabilityID: "finance.invoice.approve",
		TraceID: "trace-meta-001", IdempotencyKey: "idem-meta-1",
		ApprovalGrantID: "grant-123",
		ActorType: "user", ActorID: "actor-1",
		Action: domain.AuditActionInvoiceApprove,
		ObjectType: "invoice", ObjectID: "inv-1",
		Outcome: domain.AuditOutcomeSuccess,
		Payload: []byte(`{"ref":"inv-1","details":""}`),
	}
	if err := audit.Append(context.Background(), entry); err != nil {
		t.Fatalf("Append: %v", err)
	}

	list, err := audit.List(context.Background(), domain.AuditListQuery{EntityID: e1, Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("expected audit entry")
	}

	last := list[len(list)-1]
	if last.V2CapabilityID != "finance.invoice.approve" {
		t.Errorf("expected v2_capability_id, got %s", last.V2CapabilityID)
	}
	if last.ApprovalGrantID != "grant-123" {
		t.Errorf("expected approval_grant_id grant-123, got %s", last.ApprovalGrantID)
	}
	if last.ObjectType != "invoice" || last.ObjectID != "inv-1" {
		t.Errorf("expected object_type=invoice, object_id=inv-1, got %s/%s", last.ObjectType, last.ObjectID)
	}
}
