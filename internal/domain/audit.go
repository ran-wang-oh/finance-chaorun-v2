package domain

type AuditEntry struct {
	ID               string `json:"id"`
	EntityID         string `json:"entity_id"`
	BookID           string `json:"book_id,omitempty"`
	CapabilityID     string `json:"capability_id"`
	V2CapabilityID   string `json:"v2_capability_id,omitempty"`
	ActorType        string `json:"actor_type"`
	ActorID          string `json:"actor_id"`
	TraceID          string `json:"trace_id"`
	WorkflowRunID    string `json:"workflow_run_id,omitempty"`
	ApprovalGrantID  string `json:"approval_grant_id,omitempty"`
	IdempotencyKey   string `json:"idempotency_key,omitempty"`
	ObjectType       string `json:"object_type,omitempty"`
	ObjectID         string `json:"object_id,omitempty"`
	Action           string `json:"action"`
	Outcome          string `json:"outcome"`
	Payload          []byte `json:"payload,omitempty"`
	CreatedAt        string `json:"created_at"`
}

type AuditListQuery struct {
	EntityID string
	BookID   string
	Action   string
	Limit    int
	Offset   int
}

const (
	AuditOutcomeSuccess = "success"
	AuditOutcomeFailure = "failure"

	AuditActionInvoiceCreateDraft = "invoice.create_draft"
	AuditActionInvoiceApprove     = "invoice.approve"
	AuditActionInvoiceReject      = "invoice.reject"
	AuditActionJournalCreateDraft = "journal.create_draft"
	AuditActionJournalPost        = "journal.post"
	AuditActionJournalVoid        = "journal.void"
	AuditActionPeriodClose        = "period.close"
	AuditActionPeriodReopen       = "period.reopen"
	AuditActionPeriodLock         = "period.lock"
	AuditActionBookCreate         = "book.create"
	AuditActionAccountCreate      = "account.create"
	AuditActionAccountUpdate      = "account.update"
	AuditActionAccountDelete      = "account.delete"
	AuditActionJournalUpdate          = "journal.update_draft"
	AuditActionInvoiceUpdate          = "invoice.update_draft"
	AuditActionInvoiceImport          = "invoice.import_einvoice"
	AuditActionInvoiceVerify          = "invoice.verify"
	AuditActionInvoiceConfirmUsage    = "invoice.confirm_usage"
)
