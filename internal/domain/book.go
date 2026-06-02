package domain

type AccountingBook struct {
	ID                 string `json:"id"`
	EntityID           string `json:"entity_id"`
	Code               string `json:"code"`
	Name               string `json:"name"`
	AccountingStandard string `json:"accounting_standard"`
	BaseCurrency       string `json:"base_currency"`
	StartPeriod        string `json:"start_period"`
	IsDefault          bool   `json:"is_default"`
	Status             string `json:"status"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

const (
	BookStatusActive   = "active"
	BookStatusInactive = "inactive"
)
