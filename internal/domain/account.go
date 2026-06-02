package domain

type ChartAccount struct {
	ID           string   `json:"id"`
	EntityID     string   `json:"entity_id"`
	BookID       string   `json:"book_id"`
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	BalanceType  string   `json:"balance_type"`
	ParentID     string   `json:"parent_id,omitempty"`
	IsSystem     bool     `json:"is_system"`
	TaxRelevant  bool     `json:"tax_relevant"`
	Keywords     []string `json:"keywords,omitempty"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

const (
	BalanceTypeDebit  = "debit"
	BalanceTypeCredit = "credit"
)

type AccountListQuery struct {
	EntityID string
	Category string
	Keyword  string
	Limit    int
	Offset   int
}

type AccountListResponse struct {
	Items  []ChartAccount `json:"items"`
	Total  int            `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}
