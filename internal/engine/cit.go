package engine

import (
	"fmt"
	"math"
)

// CITInput holds the data needed for quarterly CIT prepayment calculation.
type CITInput struct {
	TaxableIncome   float64 `json:"taxable_income"`
	AccumulatedPaid float64 `json:"accumulated_paid"`
	TaxYear         string  `json:"tax_year"`
	IsSmallProfit   bool    `json:"is_small_profit"`
}

// CITOutput holds the calculated CIT.
type CITOutput struct {
	TaxableIncome   float64 `json:"taxable_income"`
	ApplicableRate  float64 `json:"applicable_rate"`
	Deduction       float64 `json:"deduction"`
	TaxPayable      float64 `json:"tax_payable"`
	AccumulatedPaid float64 `json:"accumulated_paid"`
	TaxDue          float64 `json:"tax_due"`
	EffectiveRate   float64 `json:"effective_rate"`
}

// CalcQuarterlyPrepay computes the CIT prepayment for a quarter.
func CalcQuarterlyPrepay(accumulatedProfit, applicableRate, alreadyPaid float64) (float64, error) {
	accumulatedProfit = round2(accumulatedProfit)
	alreadyPaid = round2(alreadyPaid)
	if accumulatedProfit < 0 {
		return 0, fmt.Errorf("accumulated profit must be non-negative")
	}
	if applicableRate <= 0 || applicableRate > 1 {
		return 0, fmt.Errorf("applicable rate must be between 0 and 1")
	}
	accumulatedTax := round2(accumulatedProfit * applicableRate)
	prepay := round2(accumulatedTax - alreadyPaid)
	if prepay < 0 {
		prepay = 0
	}
	return prepay, nil
}

// CalcAnnualSettlement computes the annual CIT settlement (汇算清缴).
func CalcAnnualSettlement(totalProfit, totalAdjustments, applicableRate, prepaid float64, isSmallProfit bool) (CITOutput, error) {
	totalProfit = round2(totalProfit)
	totalAdjustments = round2(totalAdjustments)
	prepaid = round2(prepaid)

	if applicableRate <= 0 || applicableRate > 1 {
		return CITOutput{}, fmt.Errorf("applicable rate must be between 0 and 1")
	}

	taxableIncome := round2(totalProfit + totalAdjustments)
	if taxableIncome < 0 {
		taxableIncome = 0
	}

	var taxPayable float64
	var effRate float64
	if isSmallProfit && taxableIncome <= 3_000_000 {
		taxPayable, effRate = CalcSmallProfitCIT(taxableIncome)
	} else {
		taxPayable = round2(taxableIncome * applicableRate)
		effRate = applicableRate
	}

	standardTax := round2(taxableIncome * applicableRate)
	deduction := round2(standardTax - taxPayable)
	if deduction < 0 {
		deduction = 0
	}

	taxDue := round2(taxPayable - prepaid)

	return CITOutput{
		TaxableIncome:   taxableIncome,
		ApplicableRate:  applicableRate,
		Deduction:       deduction,
		TaxPayable:      taxPayable,
		AccumulatedPaid: prepaid,
		TaxDue:          taxDue,
		EffectiveRate:   effRate,
	}, nil
}

// CalcSmallProfitCIT computes the CIT for small low-profit enterprises.
// <= 1,000,000: effective 5%, 1,000,001-3,000,000: effective 10%, > 3,000,000: standard 25%.
func CalcSmallProfitCIT(taxableIncome float64) (tax float64, effectiveRate float64) {
	taxableIncome = round2(taxableIncome)
	if taxableIncome <= 0 {
		return 0, 0
	}
	switch {
	case taxableIncome <= 1_000_000:
		tax = round2(taxableIncome * 0.05)
	case taxableIncome <= 3_000_000:
		tax = round2(1_000_000*0.05 + (taxableIncome-1_000_000)*0.10)
	default:
		tax = round2(taxableIncome * 0.25)
	}
	effectiveRate = math.Round(tax/taxableIncome*10000) / 10000
	return tax, effectiveRate
}
