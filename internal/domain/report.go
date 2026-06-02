package domain

type TrialBalanceRow struct {
	AccountCode  string  `json:"account_code"`
	AccountName  string  `json:"account_name"`
	DebitAmount  float64 `json:"debit_amount"`
	CreditAmount float64 `json:"credit_amount"`
	NetAmount    float64 `json:"net_amount"`
}

type TrialBalance struct {
	Period      string             `json:"period"`
	TotalDebit  float64            `json:"total_debit"`
	TotalCredit float64            `json:"total_credit"`
	Balanced    bool               `json:"balanced"`
	Rows        []TrialBalanceRow  `json:"rows"`
}

type AccountBalanceRow struct {
	AccountCode  string  `json:"account_code"`
	AccountName  string  `json:"account_name"`
	DebitAmount  float64 `json:"debit_amount"`
	CreditAmount float64 `json:"credit_amount"`
	EndingDebit  float64 `json:"ending_debit"`
	EndingCredit float64 `json:"ending_credit"`
}

type AccountBalance struct {
	Period string               `json:"period"`
	Rows   []AccountBalanceRow  `json:"rows"`
}

type VATSummary struct {
	Period               string  `json:"period"`
	InvoiceCount         int     `json:"invoice_count"`
	InputAmountWithoutTax float64 `json:"input_amount_without_tax"`
	InputTaxAmount       float64 `json:"input_tax_amount"`
	InputAmountWithTax   float64 `json:"input_amount_with_tax"`
	OutputInvoiceCount   int     `json:"output_invoice_count"`
	OutputAmountWithoutTax float64 `json:"output_amount_without_tax"`
	OutputTaxAmount      float64 `json:"output_tax_amount"`
	OutputAmountWithTax  float64 `json:"output_amount_with_tax"`
}

// ProfitStatement is the 利润表 (income statement).
type ProfitStatement struct {
	Revenue          float64               `json:"revenue"`
	Cost             float64               `json:"cost"`
	TaxAndSurcharge  float64               `json:"tax_and_surcharge"`
	SellingExpense   float64               `json:"selling_expense"`
	AdminExpense     float64               `json:"admin_expense"`
	FinanceExpense   float64               `json:"finance_expense"`
	AssetImpairment  float64               `json:"asset_impairment"`
	FairValueGain    float64               `json:"fair_value_gain"`
	InvestmentIncome float64               `json:"investment_income"`
	OperatingProfit  float64               `json:"operating_profit"`
	NonOpIncome      float64               `json:"non_op_income"`
	NonOpExpense     float64               `json:"non_op_expense"`
	TotalProfit      float64               `json:"total_profit"`
	IncomeTax        float64               `json:"income_tax"`
	NetProfit        float64               `json:"net_profit"`
	Lines            []ProfitStatementLine `json:"lines"`
}

type ProfitStatementLine struct {
	LineCode   string  `json:"line_code"`
	Label      string  `json:"label"`
	Amount     float64 `json:"amount"`
	IsSubtotal bool    `json:"is_subtotal"`
}

// BalanceSheet is the 资产负债表.
type BalanceSheet struct {
	Assets           []BalanceSheetLine `json:"assets"`
	Liabilities      []BalanceSheetLine `json:"liabilities"`
	Equity           []BalanceSheetLine `json:"equity"`
	TotalAssets      float64            `json:"total_assets"`
	TotalLiabilities float64            `json:"total_liabilities"`
	TotalEquity      float64            `json:"total_equity"`
	TotalLiabEquity  float64            `json:"total_liab_equity"`
}

type BalanceSheetLine struct {
	LineCode   string  `json:"line_code"`
	Label      string  `json:"label"`
	Amount     float64 `json:"amount"`
	IsSubtotal bool    `json:"is_subtotal"`
	Section    string  `json:"section"`
}

// VATCrossCheck compares input/output VAT rates.
type VATCrossCheck struct {
	OutputRates []VATRateRow `json:"output_rates"`
	InputRates  []VATRateRow `json:"input_rates"`
	OutputTotal float64      `json:"output_total"`
	InputTotal  float64      `json:"input_total"`
	NetPayable  float64      `json:"net_payable"`
	Warnings    []string     `json:"warnings"`
}

type VATRateRow struct {
	TaxRate         float64 `json:"tax_rate"`
	InvoiceCount    int     `json:"invoice_count"`
	AmountWithoutTax float64 `json:"amount_without_tax"`
	TaxAmount       float64 `json:"tax_amount"`
	AmountWithTax   float64 `json:"amount_with_tax"`
}

// VATReturn is the full VAT return form (金税四期).
type VATReturn struct {
	Schedule1 []VATReturnRow `json:"schedule1"`
	Schedule2 []VATReturnRow `json:"schedule2"`
	Main      VATMainTable   `json:"main"`
}

type VATMainTable struct {
	OutputTax               float64 `json:"output_tax"`
	InputTax                float64 `json:"input_tax"`
	TaxPayable              float64 `json:"tax_payable"`
	UrbanConstruction       float64 `json:"urban_construction"`
	EducationSurcharge      float64 `json:"education_surcharge"`
	LocalEducation          float64 `json:"local_education"`
	TotalTaxBurden          float64 `json:"total_tax_burden"`
}

type VATReturnRow struct {
	RowNo           int     `json:"row_no"`
	Description     string  `json:"description"`
	TaxRate         float64 `json:"tax_rate"`
	AmountWithoutTax float64 `json:"amount_without_tax"`
	TaxAmount       float64 `json:"tax_amount"`
}

// CrossTaxValidation compares VAT sales vs CIT revenue.
type CrossTaxValidation struct {
	VATSales      float64  `json:"vat_sales"`
	CITRevenue    float64  `json:"cit_revenue"`
	Difference    float64  `json:"difference"`
	DeviationRate float64  `json:"deviation_rate"`
	Consistent    bool     `json:"consistent"`
	Warnings      []string `json:"warnings"`
}

// GeneratedReport wraps generated reports for persistence.
type GeneratedReport struct {
	ID                 string           `json:"id"`
	EntityID           string           `json:"entity_id"`
	BookID             string           `json:"book_id,omitempty"`
	ReportType         string           `json:"report_type"`
	Period             string           `json:"period"`
	AccountingStandard string           `json:"accounting_standard"`
	MappingVersion     string           `json:"mapping_version,omitempty"`
	ProfitStatement    *ProfitStatement `json:"profit_statement,omitempty"`
	BalanceSheet       *BalanceSheet    `json:"balance_sheet,omitempty"`
	TrialBalance       *TrialBalance    `json:"trial_balance,omitempty"`
	CreatedAt          string           `json:"created_at,omitempty"`
}
