package domain

// RiskRule defines a risk detection rule.
type RiskRule struct {
	ID          string `json:"id"`
	RuleCode    string `json:"rule_code"`
	RuleName    string `json:"rule_name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	IsEnabled   bool   `json:"is_enabled"`
}

// RiskScan stores a risk scan result.
type RiskScan struct {
	ID            string  `json:"id"`
	EntityID      string  `json:"entity_id"`
	BookID        string  `json:"book_id"`
	Period        string  `json:"period"`
	ScanType      string  `json:"scan_type"`
	RulesTriggered []byte `json:"rules_triggered"`
	Findings      []byte  `json:"findings"`
	TotalScore    float64 `json:"total_score"`
	RiskLevel     string  `json:"risk_level"`
	EngineVersion string  `json:"engine_version,omitempty"`
	CreatedAt     string  `json:"created_at,omitempty"`
}

// Risk scan types.
const (
	ScanTypeFull     = "full"
	ScanTypeQuick    = "quick"
	ScanTypePreClose = "pre_close"
)

// RiskFinding is a single detected risk item.
type RiskFinding struct {
	RuleCode   string  `json:"rule_code"`
	Category   string  `json:"category"`
	Severity   string  `json:"severity"`
	Title      string  `json:"title"`
	Detail     string  `json:"detail"`
	Suggestion string  `json:"suggestion,omitempty"`
	Amount     float64 `json:"amount,omitempty"`
	Score      float64 `json:"score"`
}

// Risk severity levels.
const (
	SeverityBlocking = "blocking"
	SeverityDanger   = "danger"
	SeverityWarning  = "warning"
	SeverityInfo     = "info"
)

// ConsistencyCheck records a cross-source consistency check result.
type ConsistencyCheck struct {
	ID          string  `json:"id"`
	EntityID    string  `json:"entity_id"`
	BookID      string  `json:"book_id"`
	Period      string  `json:"period"`
	CheckType   string  `json:"check_type"`
	CheckName   string  `json:"check_name"`
	SourceValue float64 `json:"source_value"`
	TargetValue float64 `json:"target_value"`
	Difference  float64 `json:"difference"`
	Tolerance   float64 `json:"tolerance"`
	Passed      bool    `json:"passed"`
	Detail      string  `json:"detail,omitempty"`
	CreatedAt   string  `json:"created_at,omitempty"`
}

// TaxRiskItem represents a single tax risk finding.
type TaxRiskItem struct {
	Level      string  `json:"level"`
	Category   string  `json:"category"`
	Title      string  `json:"title"`
	Detail     string  `json:"detail"`
	Suggestion string  `json:"suggestion,omitempty"`
	Amount     float64 `json:"amount,omitempty"`
}

// TaxRiskReport is the tax risk analysis report.
type TaxRiskReport struct {
	Period      string        `json:"period"`
	Risks       []TaxRiskItem `json:"risks"`
	TotalRisks  int           `json:"total_risks"`
	DangerCount int           `json:"danger_count"`
	WarnCount   int           `json:"warn_count"`
}
