package domain

type AccountingPeriod struct {
	ID        string `json:"id"`
	EntityID  string `json:"entity_id"`
	BookID    string `json:"book_id"`
	Period    string `json:"period"`
	Status    string `json:"status"`
	OpenedAt  string `json:"opened_at,omitempty"`
	ClosingAt string `json:"closing_at,omitempty"`
	ClosedAt  string `json:"closed_at,omitempty"`
	LockedAt  string `json:"locked_at,omitempty"`
	ClosedBy  string `json:"closed_by,omitempty"`
	Metadata  []byte `json:"metadata,omitempty"`
}

const (
	PeriodStatusOpen    = "open"
	PeriodStatusClosing = "closing"
	PeriodStatusClosed  = "closed"
	PeriodStatusLocked  = "locked"
)

type CloseCheck struct {
	Category string `json:"category"`
	Label    string `json:"label"`
	Passed   bool   `json:"passed"`
	Detail   string `json:"detail"`
	Severity string `json:"severity"`
}

type CloseCheckResult struct {
	Passed  bool         `json:"passed"`
	Period  string       `json:"period"`
	Checks  []CloseCheck `json:"checks"`
	Summary string       `json:"summary"`
}
