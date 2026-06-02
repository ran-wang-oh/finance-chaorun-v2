package engine

import (
	"fmt"
	"sort"
	"strings"
)

// ---- Zero-filing detection ----

// ZeroFilingRisk flags consecutive months with no tax payable.
type ZeroFilingRisk struct {
	ConsecutiveMonths int      `json:"consecutive_months"`
	Periods           []string `json:"periods"`
	Level             string   `json:"level"`
}

// DetectZeroFiling scans monthly tax payable data for consecutive zero-filing runs.
func DetectZeroFiling(periods []string, taxPayable map[string]float64) []ZeroFilingRisk {
	if len(periods) == 0 {
		return nil
	}
	sorted := make([]string, len(periods))
	copy(sorted, periods)
	sort.Strings(sorted)

	var risks []ZeroFilingRisk
	var run []string
	for _, p := range sorted {
		v, ok := taxPayable[p]
		if ok && v == 0 {
			run = append(run, p)
		} else {
			if len(run) >= 2 {
				lvl := "warning"
				if len(run) >= 3 {
					lvl = "danger"
				}
				risks = append(risks, ZeroFilingRisk{
					ConsecutiveMonths: len(run),
					Periods:           append([]string{}, run...),
					Level:             lvl,
				})
			}
			run = nil
		}
	}
	if len(run) >= 2 {
		lvl := "warning"
		if len(run) >= 3 {
			lvl = "danger"
		}
		risks = append(risks, ZeroFilingRisk{
			ConsecutiveMonths: len(run),
			Periods:           append([]string{}, run...),
			Level:             lvl,
		})
	}
	return risks
}

// ---- Consecutive loss detection ----

// ConsecutiveLossRisk flags consecutive years of operating losses.
type ConsecutiveLossRisk struct {
	ConsecutiveYears int   `json:"consecutive_years"`
	Years            []int `json:"years"`
	Level            string `json:"level"`
}

// DetectConsecutiveLosses scans annual operating profit for consecutive loss years.
func DetectConsecutiveLosses(years []int, operatingProfit map[int]float64) []ConsecutiveLossRisk {
	if len(years) == 0 {
		return nil
	}
	sort.Ints(years)

	var risks []ConsecutiveLossRisk
	var run []int
	for _, y := range years {
		v, ok := operatingProfit[y]
		if ok && v < 0 {
			run = append(run, y)
		} else {
			if len(run) >= 2 {
				lvl := "warning"
				if len(run) >= 3 {
					lvl = "danger"
				}
				risks = append(risks, ConsecutiveLossRisk{
					ConsecutiveYears: len(run),
					Years:            append([]int{}, run...),
					Level:            lvl,
				})
			}
			run = nil
		}
	}
	if len(run) >= 2 {
		lvl := "warning"
		if len(run) >= 3 {
			lvl = "danger"
		}
		risks = append(risks, ConsecutiveLossRisk{
			ConsecutiveYears: len(run),
			Years:            append([]int{}, run...),
			Level:            lvl,
		})
	}
	return risks
}

// ---- Tax burden benchmark ----

// TaxBurdenDeviation reports how much effective tax burden deviates from industry benchmark.
type TaxBurdenDeviation struct {
	EffectiveRate  float64 `json:"effective_rate"`
	IndustryAvg    float64 `json:"industry_avg"`
	DeviationRatio float64 `json:"deviation_ratio"`
	Level          string  `json:"level"`
}

// Industry tax burden benchmarks.
const (
	IndustryAvgManufacturing = 0.030
	IndustryAvgWholesale     = 0.008
	IndustryAvgServices      = 0.025
	IndustryAvgConstruction  = 0.020
	IndustryAvgDefault       = 0.020
)

// CalcTaxBurdenBenchmark compares the effective tax burden rate against the industry average.
func CalcTaxBurdenBenchmark(effectiveRate float64, industry string, taxpayerType string) *TaxBurdenDeviation {
	var industryAvg float64
	switch strings.ToLower(industry) {
	case "manufacturing", "制造业":
		industryAvg = IndustryAvgManufacturing
	case "wholesale", "批发零售", "批发", "零售":
		industryAvg = IndustryAvgWholesale
	case "services", "服务业", "信息技术", "软件":
		industryAvg = IndustryAvgServices
	case "construction", "建筑业", "建筑":
		industryAvg = IndustryAvgConstruction
	default:
		industryAvg = IndustryAvgDefault
	}

	if taxpayerType == "small_scale" || taxpayerType == "小规模纳税人" {
		industryAvg = industryAvg * 0.5
	}

	if effectiveRate <= 0 || industryAvg <= 0 {
		return nil
	}

	ratio := round2(effectiveRate / industryAvg)
	level := "info"
	if ratio < 0.4 {
		level = "danger"
	} else if ratio < 0.6 {
		level = "warning"
	} else {
		return nil
	}

	return &TaxBurdenDeviation{
		EffectiveRate:  effectiveRate,
		IndustryAvg:    industryAvg,
		DeviationRatio: ratio,
		Level:          level,
	}
}

// FormatZeroFilingRisk converts ZeroFilingRisk to a human-readable Chinese string.
func FormatZeroFilingRisk(risk ZeroFilingRisk) string {
	periods := strings.Join(risk.Periods, ", ")
	return fmt.Sprintf("连续 %d 个月零申报（%s），易触发税务核查。请确认是否有未申报收入。",
		risk.ConsecutiveMonths, periods)
}

// FormatConsecutiveLossRisk converts ConsecutiveLossRisk to a human-readable Chinese string.
func FormatConsecutiveLossRisk(risk ConsecutiveLossRisk) string {
	yearStrs := make([]string, len(risk.Years))
	for i, y := range risk.Years {
		yearStrs[i] = fmt.Sprintf("%d", y)
	}
	return fmt.Sprintf("连续 %d 年亏损（%s），连续亏损超过 2 年将触发税务约谈。请核实亏损合理性。",
		risk.ConsecutiveYears, strings.Join(yearStrs, ", "))
}

// FormatTaxBurdenDeviation converts TaxBurdenDeviation to a human-readable Chinese string.
func FormatTaxBurdenDeviation(d *TaxBurdenDeviation) string {
	return fmt.Sprintf("实际税负率 %.1f%%，仅为行业均值 %.1f%% 的 %.0f%%，显著偏低，可能触发税务核查。",
		d.EffectiveRate*100, d.IndustryAvg*100, d.DeviationRatio*100)
}

// ---- Invoice-revenue mismatch detection ----

// InvoiceRevenueMismatchRisk flags when invoice-derived revenue deviates from declared revenue.
type InvoiceRevenueMismatchRisk struct {
	InvoiceRevenue  float64 `json:"invoice_revenue"`
	DeclaredRevenue float64 `json:"declared_revenue"`
	Difference      float64 `json:"difference"`
	DeviationRate   float64 `json:"deviation_rate"`
	Level           string  `json:"level"`
}

// DetectInvoiceRevenueMismatch compares invoice-derived revenue against declared revenue.
func DetectInvoiceRevenueMismatch(invoiceRevenue, declaredRevenue float64) *InvoiceRevenueMismatchRisk {
	if declaredRevenue <= 0 {
		return nil
	}
	diff := round2(invoiceRevenue - declaredRevenue)
	rate := round2(diff / declaredRevenue)
	absRate := rate
	if absRate < 0 {
		absRate = -absRate
	}
	if absRate <= 0.05 {
		return nil
	}
	level := "warning"
	if absRate > 0.15 {
		level = "danger"
	}
	return &InvoiceRevenueMismatchRisk{
		InvoiceRevenue:  invoiceRevenue,
		DeclaredRevenue: declaredRevenue,
		Difference:      diff,
		DeviationRate:   rate,
		Level:           level,
	}
}

// ---- Expense-to-revenue ratio anomaly ----

// ExpenseRatioRisk flags when total expense/revenue ratio exceeds normal bounds.
type ExpenseRatioRisk struct {
	TotalExpense float64 `json:"total_expense"`
	Revenue      float64 `json:"revenue"`
	Ratio        float64 `json:"ratio"`
	IndustryAvg  float64 `json:"industry_avg"`
	Level        string  `json:"level"`
}

// DetectExpenseToRevenueRatio checks if the expense/revenue ratio is abnormal for the industry.
func DetectExpenseToRevenueRatio(totalExpense, revenue float64, industry string) *ExpenseRatioRisk {
	if revenue <= 0 {
		return nil
	}
	ratio := round2(totalExpense / revenue)

	var industryAvg float64
	switch strings.ToLower(industry) {
	case "manufacturing", "制造业":
		industryAvg = 0.75
	case "wholesale", "批发零售", "批发", "零售":
		industryAvg = 0.92
	case "services", "服务业", "信息技术", "软件":
		industryAvg = 0.85
	case "construction", "建筑业", "建筑":
		industryAvg = 0.80
	default:
		industryAvg = 0.85
	}

	if ratio <= industryAvg {
		return nil
	}
	level := "warning"
	if ratio > 1.0 {
		level = "danger"
	}
	return &ExpenseRatioRisk{
		TotalExpense: totalExpense,
		Revenue:      revenue,
		Ratio:        ratio,
		IndustryAvg:  industryAvg,
		Level:        level,
	}
}

// ---- Large round transactions ----

// RoundTransactionRisk flags invoices with rounded amounts that may indicate circular invoicing.
type RoundTransactionRisk struct {
	InvoiceNo string  `json:"invoice_no"`
	Amount    float64 `json:"amount"`
	IsRounded bool    `json:"is_rounded"`
	Level     string  `json:"level"`
}

// RoundTransactionInput holds invoice data for round-transaction detection.
type RoundTransactionInput struct {
	InvoiceNo     string
	AmountWithTax float64
}

// DetectLargeRoundTransactions flags invoices where the tax-inclusive amount is
// a round multiple of 10,000 CNY and exceeds 50,000 CNY.
func DetectLargeRoundTransactions(invoices []RoundTransactionInput) []RoundTransactionRisk {
	var risks []RoundTransactionRisk
	for _, inv := range invoices {
		if inv.AmountWithTax < 50000 {
			continue
		}
		mod := inv.AmountWithTax - float64(int(inv.AmountWithTax/10000))*10000
		if mod < 0 {
			mod = -mod
		}
		if mod < 0.01 {
			risks = append(risks, RoundTransactionRisk{
				InvoiceNo: inv.InvoiceNo,
				Amount:    inv.AmountWithTax,
				IsRounded: true,
				Level:     "info",
			})
		}
	}
	return risks
}

// ---- Supplier concentration ----

// SupplierConcentrationRisk flags when a single supplier dominates input invoices.
type SupplierConcentrationRisk struct {
	TopSupplier   string  `json:"top_supplier"`
	TopAmount     float64 `json:"top_amount"`
	TotalInput    float64 `json:"total_input"`
	Concentration float64 `json:"concentration"`
	Level         string  `json:"level"`
}

// DetectSupplierConcentration checks if any single supplier accounts for too much input.
func DetectSupplierConcentration(sellerAmounts map[string]float64) *SupplierConcentrationRisk {
	if len(sellerAmounts) == 0 {
		return nil
	}
	var topSupplier string
	var topAmount, total float64
	for seller, amt := range sellerAmounts {
		total += amt
		if amt > topAmount {
			topAmount = amt
			topSupplier = seller
		}
	}
	if total <= 0 || topAmount <= 0 {
		return nil
	}
	concentration := round2(topAmount / total)
	if concentration < 0.50 {
		return nil
	}
	level := "warning"
	if concentration > 0.70 {
		level = "danger"
	}
	return &SupplierConcentrationRisk{
		TopSupplier:   topSupplier,
		TopAmount:     topAmount,
		TotalInput:    total,
		Concentration: concentration,
		Level:         level,
	}
}

// ---- Sudden revenue change ----

// SuddenRevenueChangeRisk flags large period-over-period revenue swings.
type SuddenRevenueChangeRisk struct {
	CurrentRevenue  float64 `json:"current_revenue"`
	PreviousRevenue float64 `json:"previous_revenue"`
	ChangeRate      float64 `json:"change_rate"`
	Direction       string  `json:"direction"`
	Level           string  `json:"level"`
}

// DetectSuddenRevenueChange flags period-over-period revenue changes exceeding 50%.
func DetectSuddenRevenueChange(currentRevenue, previousRevenue float64) *SuddenRevenueChangeRisk {
	if previousRevenue <= 0 {
		return nil
	}
	change := round2((currentRevenue - previousRevenue) / previousRevenue)
	absChange := change
	if absChange < 0 {
		absChange = -absChange
	}
	if absChange <= 0.50 {
		return nil
	}
	dir := "surge"
	if change < 0 {
		dir = "drop"
	}
	level := "warning"
	if absChange > 1.0 {
		level = "danger"
	}
	return &SuddenRevenueChangeRisk{
		CurrentRevenue:  currentRevenue,
		PreviousRevenue: previousRevenue,
		ChangeRate:      change,
		Direction:       dir,
		Level:           level,
	}
}

// ---- Risk scoring ----

// RiskFinding is a single risk detection result with a score contribution.
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

// ScoreRiskFindings computes a composite risk score and risk level.
func ScoreRiskFindings(findings []RiskFinding) (float64, string) {
	var total float64
	for _, f := range findings {
		switch f.Severity {
		case "blocking":
			total += 25
		case "danger":
			total += 15
		case "warning":
			total += 5
		case "info":
			total += 1
		default:
			total += 1
		}
	}
	var level string
	switch {
	case total > 50:
		level = "critical"
	case total > 30:
		level = "high"
	case total > 10:
		level = "medium"
	default:
		level = "low"
	}
	return total, level
}
