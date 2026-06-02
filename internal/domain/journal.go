package domain

type JournalEntry struct {
	ID          string        `json:"id"`
	EntityID    string        `json:"entity_id"`
	BookID      string        `json:"book_id"`
	Period      string        `json:"period"`
	VoucherNo   string        `json:"voucher_no,omitempty"`
	VoucherWord string        `json:"voucher_word"`
	EntryDate   string        `json:"entry_date"`
	Summary     string        `json:"summary"`
	SourceType  string        `json:"source_type,omitempty"`
	SourceID    string        `json:"source_id,omitempty"`
	Status      string        `json:"status"`
	CreatedBy   string        `json:"created_by,omitempty"`
	PostedBy    string        `json:"posted_by,omitempty"`
	PostedAt    string        `json:"posted_at,omitempty"`
	Lines       []JournalLine `json:"lines"`
	CreatedAt   string        `json:"created_at,omitempty"`
	UpdatedAt   string        `json:"updated_at,omitempty"`
}

type JournalLine struct {
	ID              string  `json:"id,omitempty"`
	EntityID        string  `json:"entity_id"`
	JournalEntryID  string  `json:"journal_entry_id"`
	AccountID       string  `json:"account_id"`
	AccountCode     string  `json:"account_code"`
	AccountName     string  `json:"account_name"`
	Direction       string  `json:"direction"`
	DebitAmount     float64 `json:"debit_amount"`
	CreditAmount    float64 `json:"credit_amount"`
	Currency        string  `json:"currency"`
	LineNo          int     `json:"line_no"`
	Auxiliary       []byte  `json:"auxiliary,omitempty"`
}

const (
	JournalStatusDraft  = "draft"
	JournalStatusPosted = "posted"
	JournalStatusVoided = "voided"

	DirectionDebit  = "debit"
	DirectionCredit = "credit"

	VoucherWordJi = "记"
)

type JournalListQuery struct {
	EntityID string
	BookID   string
	Status   string
	Period   string
	Limit    int
	Offset   int
}

type JournalListResponse struct {
	Items  []JournalEntry `json:"items"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}
