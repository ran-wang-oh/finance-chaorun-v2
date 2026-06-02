package domain

// Adjustment categories.
const (
	AdjCatEntertainment = "entertainment"
	AdjCatAdvertising   = "advertising"
	AdjCatWelfare       = "welfare"
	AdjCatDepreciation  = "depreciation"
	AdjCatImpairment    = "impairment"
	AdjCatRDDeduction   = "rd_deduction"
	AdjCatOther         = "other"
)

// AdjustmentRecord is a tax adjustment for CIT purposes.
type AdjustmentRecord struct {
	ID         string  `json:"id"`
	EntityID   string  `json:"entity_id"`
	BookID     string  `json:"book_id"`
	TaxYear    string  `json:"tax_year"`
	Category   string  `json:"category"`
	BookAmount float64 `json:"book_amount"`
	TaxBase    float64 `json:"tax_base"`
	Adjustment float64 `json:"adjustment"`
	Formula    string  `json:"formula,omitempty"`
	Detail     string  `json:"detail,omitempty"`
	CreatedAt  string  `json:"created_at,omitempty"`
}

// AdjustmentListQuery filters adjustments.
type AdjustmentListQuery struct {
	EntityID string
	TaxYear  string
	Category string
	Limit    int
	Offset   int
}

// TaxProfile holds entity/book tax registration info.
type TaxProfile struct {
	ID                string `json:"id"`
	EntityID          string `json:"entity_id"`
	BookID            string `json:"book_id"`
	TaxpayerType      string `json:"taxpayer_type"`
	VATTaxpayerType   string `json:"vat_taxpayer_type"`
	CITRateType       string `json:"cit_rate_type"`
	TaxRegistrationNo string `json:"tax_registration_no"`
	TaxOffice         string `json:"tax_office,omitempty"`
	Industry          string `json:"industry,omitempty"`
	CreatedAt         string `json:"created_at,omitempty"`
	UpdatedAt         string `json:"updated_at,omitempty"`
}

// Taxpayer type constants.
const (
	TaxpayerGeneral    = "general"
	TaxpayerSmallScale = "small_scale"
)

// CIT rate type constants.
const (
	CITRateStandard    = "standard"
	CITRateSmallProfit = "small_profit"
	CITRateHighTech    = "high_tech"
	CITRateWestern     = "western"
)

// TaxReturn stores a generated tax return.
type TaxReturn struct {
	ID         string `json:"id"`
	EntityID   string `json:"entity_id"`
	BookID     string `json:"book_id"`
	TaxYear    string `json:"tax_year"`
	TaxPeriod  string `json:"tax_period"`
	ReturnType string `json:"return_type"`
	Payload    []byte `json:"payload"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at,omitempty"`
}

// Tax return types.
const (
	ReturnTypeVAT       = "vat"
	ReturnTypeCIT       = "cit"
	ReturnTypeSurcharges = "surcharges"
	ReturnTypeStampTax  = "stamp_tax"
	ReturnTypePITSalary = "pit_salary"
)

// CITReport is a generated CIT annual report.
type CITReport struct {
	ID             string  `json:"id"`
	EntityID       string  `json:"entity_id"`
	BookID         string  `json:"book_id"`
	TaxYear        string  `json:"tax_year"`
	ReportType     string  `json:"report_type"`
	TotalRevenue   float64 `json:"total_revenue"`
	TotalCost      float64 `json:"total_cost"`
	TotalExpense   float64 `json:"total_expense"`
	OperatingProfit float64 `json:"operating_profit"`
	AdjustmentTotal float64 `json:"adjustment_total"`
	TaxableIncome  float64 `json:"taxable_income"`
	ApplicableRate float64 `json:"applicable_rate"`
	TaxPayable     float64 `json:"tax_payable"`
	Deduction      float64 `json:"deduction"`
	PrepaidTax     float64 `json:"prepaid_tax"`
	TaxDue         float64 `json:"tax_due"`
	Status         string  `json:"status"`
	GeneratedBy    string  `json:"generated_by,omitempty"`
	CreatedAt      string  `json:"created_at,omitempty"`
}
