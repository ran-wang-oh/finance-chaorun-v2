package engine

import (
	"fmt"
	"math"
)

// ConsistencyResult holds the output of a single cross-source consistency check.
type ConsistencyResult struct {
	CheckType   string  `json:"check_type"`
	CheckName   string  `json:"check_name"`
	SourceValue float64 `json:"source_value"`
	TargetValue float64 `json:"target_value"`
	Difference  float64 `json:"difference"`
	Tolerance   float64 `json:"tolerance"`
	Passed      bool    `json:"passed"`
	Detail      string  `json:"detail"`
}

// CheckInvoiceToVAT compares posted invoice tax totals against VAT return totals.
func CheckInvoiceToVAT(invoiceOutputTax, invoiceInputTax, vatOutputTax, vatInputTax float64) ConsistencyResult {
	diffOutput := round2(invoiceOutputTax - vatOutputTax)
	diffInput := round2(invoiceInputTax - vatInputTax)
	tolerance := round2(math.Max(vatOutputTax, vatInputTax) * 0.01)
	if tolerance < 1.0 {
		tolerance = 1.0
	}
	totalDiff := round2(math.Abs(diffOutput) + math.Abs(diffInput))
	passed := totalDiff <= tolerance
	return ConsistencyResult{
		CheckType:   "invoice_to_vat",
		CheckName:   "发票税额与VAT申报一致性",
		SourceValue: invoiceOutputTax,
		TargetValue: vatOutputTax,
		Difference:  totalDiff,
		Tolerance:   tolerance,
		Passed:      passed,
		Detail:      fmt.Sprintf("发票销项 %.2f vs VAT销项 %.2f (差%.2f), 发票进项 %.2f vs VAT进项 %.2f (差%.2f)", invoiceOutputTax, vatOutputTax, diffOutput, invoiceInputTax, vatInputTax, diffInput),
	}
}

// CheckInvoiceToJournal checks that approved invoice count/amount has corresponding journal entries.
func CheckInvoiceToJournal(approvedCount, journalCount int, approvedAmount, journalAmount float64) ConsistencyResult {
	countDiff := approvedCount - journalCount
	amountDiff := round2(approvedAmount - journalAmount)
	passed := countDiff == 0 && math.Abs(amountDiff) < 0.02
	detail := "发票与凭证完全匹配"
	if !passed {
		detail = fmt.Sprintf("已审批发票 %d 张 (%.2f)，对应凭证 %d 张 (%.2f)", approvedCount, approvedAmount, journalCount, journalAmount)
	}
	return ConsistencyResult{
		CheckType:   "invoice_to_journal",
		CheckName:   "发票与凭证一致性",
		SourceValue: float64(approvedCount),
		TargetValue: float64(journalCount),
		Difference:  amountDiff,
		Tolerance:   0.02,
		Passed:      passed,
		Detail:      detail,
	}
}

// CheckVATToCITRevenue compares VAT output revenue against CIT declared revenue.
func CheckVATToCITRevenue(vatRevenue, citRevenue float64) ConsistencyResult {
	diff := round2(vatRevenue - citRevenue)
	tolerance := round2(math.Max(math.Abs(vatRevenue), math.Abs(citRevenue)) * 0.05)
	if tolerance < 1.0 {
		tolerance = 1.0
	}
	passed := math.Abs(diff) <= tolerance
	return ConsistencyResult{
		CheckType:   "vat_to_cit_revenue",
		CheckName:   "增值税与所得税收入一致性",
		SourceValue: vatRevenue,
		TargetValue: citRevenue,
		Difference:  diff,
		Tolerance:   tolerance,
		Passed:      passed,
		Detail:      fmt.Sprintf("增值税收入 %.2f，所得税收入 %.2f，差异 %.2f (容忍度 %.2f)", vatRevenue, citRevenue, diff, tolerance),
	}
}

// CheckJournalToBank checks that journaled bank transactions reconcile to bank balance.
func CheckJournalToBank(journalAmount, bankAmount float64) ConsistencyResult {
	diff := round2(journalAmount - bankAmount)
	tolerance := round2(math.Max(math.Abs(journalAmount), math.Abs(bankAmount)) * 0.01)
	if tolerance < 0.02 {
		tolerance = 0.02
	}
	passed := math.Abs(diff) <= tolerance
	return ConsistencyResult{
		CheckType:   "journal_to_bank",
		CheckName:   "凭证与银行对账一致性",
		SourceValue: journalAmount,
		TargetValue: bankAmount,
		Difference:  diff,
		Tolerance:   tolerance,
		Passed:      passed,
		Detail:      fmt.Sprintf("凭证银行科目合计 %.2f，银行交易合计 %.2f，差异 %.2f", journalAmount, bankAmount, diff),
	}
}
