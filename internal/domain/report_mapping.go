package domain

// ReportMapping defines one line on a financial report.
type ReportMapping struct {
	ID                 string          `json:"id"`
	ReportType         string          `json:"report_type"`
	LineCode           string          `json:"line_code"`
	LineLabel          string          `json:"line_label"`
	DisplayOrder       int             `json:"display_order"`
	AccountingStandard string          `json:"accounting_standard"`
	AccountSelector    AccountSelector `json:"account_selector"`
	IsSubtotal         bool            `json:"is_subtotal"`
	ParentLineCode     string          `json:"parent_line_code,omitempty"`
	Formula            string          `json:"formula,omitempty"`
}

// AccountSelector defines which accounts map to a report line.
type AccountSelector struct {
	Prefixes        []string `json:"prefixes,omitempty"`
	Categories      []string `json:"categories,omitempty"`
	Direction       string   `json:"direction"`
	ExcludePrefixes []string `json:"exclude_prefixes,omitempty"`
}

const (
	ReportTypeProfitStatement = "profit_statement"
	ReportTypeBalanceSheet    = "balance_sheet"
)
