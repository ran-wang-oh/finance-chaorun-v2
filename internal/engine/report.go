package engine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ---- Report mapping types ----

// AccountSelector defines which accounts map to a report line.
type AccountSelector struct {
	Prefixes        []string `json:"prefixes,omitempty"`
	Categories      []string `json:"categories,omitempty"`
	Direction       string   `json:"direction"`
	ExcludePrefixes []string `json:"exclude_prefixes,omitempty"`
}

// ReportMapping defines one line on a financial report.
type ReportMapping struct {
	LineCode          string          `json:"line_code"`
	LineLabel         string          `json:"line_label"`
	DisplayOrder      int             `json:"display_order"`
	AccountingStandard string         `json:"accounting_standard"`
	AccountSelector   AccountSelector `json:"account_selector"`
	IsSubtotal        bool            `json:"is_subtotal"`
	ParentLineCode    string          `json:"parent_line_code,omitempty"`
	Formula           string          `json:"formula,omitempty"`
}

const (
	ReportTypeProfitStatement = "profit_statement"
	ReportTypeBalanceSheet    = "balance_sheet"
)

// ReportLine is a simplified journal line for report building.
type ReportLine struct {
	AccountCode string
	Direction   string
	Amount      float64
}

// ---- Report output types ----

// ProfitStatement is the 利润表 (income statement).
type ProfitStatement struct {
	Revenue          float64                `json:"revenue"`
	Cost             float64                `json:"cost"`
	TaxAndSurcharge  float64                `json:"tax_and_surcharge"`
	SellingExpense   float64                `json:"selling_expense"`
	AdminExpense     float64                `json:"admin_expense"`
	FinanceExpense   float64                `json:"finance_expense"`
	AssetImpairment  float64                `json:"asset_impairment"`
	FairValueGain    float64                `json:"fair_value_gain"`
	InvestmentIncome float64                `json:"investment_income"`
	OperatingProfit  float64                `json:"operating_profit"`
	NonOpIncome      float64                `json:"non_op_income"`
	NonOpExpense     float64                `json:"non_op_expense"`
	TotalProfit      float64                `json:"total_profit"`
	IncomeTax        float64                `json:"income_tax"`
	NetProfit        float64                `json:"net_profit"`
	Lines            []ProfitStatementLine  `json:"lines"`
}

// ProfitStatementLine is one row of the profit statement.
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

// BalanceSheetLine is one row of the balance sheet.
type BalanceSheetLine struct {
	LineCode   string  `json:"line_code"`
	Label      string  `json:"label"`
	Amount     float64 `json:"amount"`
	IsSubtotal bool    `json:"is_subtotal"`
	Section    string  `json:"section"`
}

// ---- ReportBuilder ----

// ReportBuilder builds financial reports from mappings and journal lines.
type ReportBuilder struct {
	mappings []ReportMapping
	byCode   map[string]ReportMapping
}

// NewReportBuilder creates a ReportBuilder. Returns nil if no mappings configured.
func NewReportBuilder(mappings []ReportMapping) *ReportBuilder {
	if len(mappings) == 0 {
		return nil
	}
	sort.Slice(mappings, func(i, j int) bool {
		return mappings[i].DisplayOrder < mappings[j].DisplayOrder
	})
	rb := &ReportBuilder{
		mappings: mappings,
		byCode:   make(map[string]ReportMapping, len(mappings)),
	}
	for _, m := range mappings {
		rb.byCode[m.LineCode] = m
	}
	return rb
}

// BuildProfitStatement applies report mappings to journal lines and returns a profit statement.
func (rb *ReportBuilder) BuildProfitStatement(lines []ReportLine) (*ProfitStatement, error) {
	if rb == nil {
		return nil, fmt.Errorf("no report mappings configured")
	}

	amounts := make(map[string]float64)
	for _, l := range lines {
		for _, m := range rb.mappings {
			if m.IsSubtotal {
				continue
			}
			if rb.matchSelector(l.AccountCode, l.Direction, m.AccountSelector) {
				amounts[m.LineCode] = round2(amounts[m.LineCode] + l.Amount)
			}
		}
	}

	evaluated := make(map[string]float64)
	for k, v := range amounts {
		evaluated[k] = v
	}

	for _, m := range rb.mappings {
		if !m.IsSubtotal || m.Formula == "" {
			continue
		}
		val, err := rb.evalFormula(m.Formula, evaluated)
		if err != nil {
			return nil, fmt.Errorf("evaluate %s (%s): %w", m.LineCode, m.Formula, err)
		}
		evaluated[m.LineCode] = val
	}

	ps := &ProfitStatement{
		Lines: make([]ProfitStatementLine, 0, len(rb.mappings)),
	}
	for _, m := range rb.mappings {
		amt := evaluated[m.LineCode]
		ps.Lines = append(ps.Lines, ProfitStatementLine{
			LineCode:   m.LineCode,
			Label:      m.LineLabel,
			Amount:     amt,
			IsSubtotal: m.IsSubtotal,
		})
		switch m.LineCode {
		case "revenue":
			ps.Revenue = amt
		case "cost":
			ps.Cost = amt
		case "tax_surcharge":
			ps.TaxAndSurcharge = amt
		case "selling_expense":
			ps.SellingExpense = amt
		case "admin_expense":
			ps.AdminExpense = amt
		case "finance_expense":
			ps.FinanceExpense = amt
		case "asset_impairment":
			ps.AssetImpairment = amt
		case "fair_value_gain":
			ps.FairValueGain = amt
		case "investment_income":
			ps.InvestmentIncome = amt
		case "operating_profit":
			ps.OperatingProfit = amt
		case "non_op_income":
			ps.NonOpIncome = amt
		case "non_op_expense":
			ps.NonOpExpense = amt
		case "total_profit":
			ps.TotalProfit = amt
		case "income_tax":
			ps.IncomeTax = amt
		case "net_profit":
			ps.NetProfit = amt
		}
	}
	return ps, nil
}

// BuildBalanceSheet applies report mappings to journal lines and returns a balance sheet.
func (rb *ReportBuilder) BuildBalanceSheet(lines []ReportLine) (*BalanceSheet, error) {
	if rb == nil {
		return nil, fmt.Errorf("no report mappings configured")
	}

	amounts := make(map[string]float64)
	for _, l := range lines {
		for _, m := range rb.mappings {
			if m.IsSubtotal {
				continue
			}
			if rb.matchSelector(l.AccountCode, l.Direction, m.AccountSelector) {
				amounts[m.LineCode] = round2(amounts[m.LineCode] + l.Amount)
			}
		}
	}

	evaluated := make(map[string]float64)
	for k, v := range amounts {
		evaluated[k] = v
	}

	for _, m := range rb.mappings {
		if !m.IsSubtotal || m.Formula == "" {
			continue
		}
		val, err := rb.evalFormula(m.Formula, evaluated)
		if err != nil {
			return nil, fmt.Errorf("evaluate %s (%s): %w", m.LineCode, m.Formula, err)
		}
		evaluated[m.LineCode] = val
	}

	bs := &BalanceSheet{
		Assets:      make([]BalanceSheetLine, 0),
		Liabilities: make([]BalanceSheetLine, 0),
		Equity:      make([]BalanceSheetLine, 0),
	}

	sectionOrder := map[string]string{
		"cash": "assets", "receivables": "assets", "prepayments": "assets",
		"other_receivables": "assets", "inventory": "assets",
		"fixed_assets_net": "assets", "intangibles_net": "assets",
		"total_assets": "assets",
		"short_term_loans": "liabilities", "payables": "liabilities",
		"advance_receipts": "liabilities", "employee_payable": "liabilities",
		"tax_payable": "liabilities", "other_payables": "liabilities",
		"total_liabilities": "liabilities",
		"paid_in_capital": "equity", "capital_reserve": "equity",
		"surplus_reserve": "equity", "retained_earnings": "equity",
		"total_equity": "equity", "total_liab_equity": "equity",
	}

	for _, m := range rb.mappings {
		amt := evaluated[m.LineCode]
		section := sectionOrder[m.LineCode]
		line := BalanceSheetLine{
			LineCode:   m.LineCode,
			Label:      m.LineLabel,
			Amount:     amt,
			IsSubtotal: m.IsSubtotal,
			Section:    section,
		}
		switch section {
		case "assets":
			bs.Assets = append(bs.Assets, line)
		case "liabilities":
			bs.Liabilities = append(bs.Liabilities, line)
		case "equity":
			bs.Equity = append(bs.Equity, line)
		}
		switch m.LineCode {
		case "total_assets":
			bs.TotalAssets = amt
		case "total_liabilities":
			bs.TotalLiabilities = amt
		case "total_equity":
			bs.TotalEquity = amt
		case "total_liab_equity":
			bs.TotalLiabEquity = amt
		}
	}
	return bs, nil
}

func (rb *ReportBuilder) matchSelector(code, direction string, sel AccountSelector) bool {
	if sel.Direction != "" && sel.Direction != "either" && sel.Direction != direction {
		return false
	}
	for _, p := range sel.ExcludePrefixes {
		if strings.HasPrefix(code, p) {
			return false
		}
	}
	if len(sel.Prefixes) > 0 {
		matched := false
		for _, p := range sel.Prefixes {
			if strings.HasPrefix(code, p) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func (rb *ReportBuilder) evalFormula(formula string, amounts map[string]float64) (float64, error) {
	formula = strings.TrimSpace(formula)
	if formula == "" {
		return 0, nil
	}

	tokens := strings.Fields(formula)
	if len(tokens) == 0 {
		return 0, nil
	}

	resolve := func(token string) float64 {
		if v, ok := amounts[token]; ok {
			return v
		}
		if f, err := strconv.ParseFloat(token, 64); err == nil {
			return f
		}
		return 0
	}

	val := resolve(tokens[0])

	for i := 1; i < len(tokens); i += 2 {
		if i+1 >= len(tokens) {
			break
		}
		op := tokens[i]
		right := resolve(tokens[i+1])
		switch op {
		case "+":
			val = round2(val + right)
		case "-":
			val = round2(val - right)
		default:
			return 0, fmt.Errorf("unsupported operator: %s", op)
		}
	}

	return val, nil
}
