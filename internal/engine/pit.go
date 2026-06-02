package engine

import "fmt"

// PIT deduction constants (个人所得税法 2019+).
const (
	PITBasicDeductionPerMonth  = 5000.00
	PITChildrenEducation       = 2000.00
	PITContinuingEducation     = 400.00
	PITHousingLoanInterest     = 1000.00
	PITHousingRentTier1        = 1500.00
	PITHousingRentTier2        = 1100.00
	PITHousingRentTier3        = 800.00
	PITElderlySupportOnlyChild = 3000.00
	PITElderlySupportSharedMax = 1500.00
	PITInfantCare              = 2000.00
)

// pitBracket returns the rate and quick deduction for cumulative taxable income.
func pitBracket(cumulativeTaxableIncome float64) (rate float64, quickDeduction float64) {
	switch {
	case cumulativeTaxableIncome <= 36000:
		return 0.03, 0
	case cumulativeTaxableIncome <= 144000:
		return 0.10, 2520
	case cumulativeTaxableIncome <= 300000:
		return 0.20, 16920
	case cumulativeTaxableIncome <= 420000:
		return 0.25, 31920
	case cumulativeTaxableIncome <= 660000:
		return 0.30, 52920
	case cumulativeTaxableIncome <= 960000:
		return 0.35, 85920
	default:
		return 0.45, 181920
	}
}

// PITInput holds the data for monthly PIT withholding calculation.
type PITInput struct {
	MonthlySalary                float64                `json:"monthly_salary"`
	AccumulatedSalary            float64                `json:"accumulated_salary"`
	Months                       int                    `json:"months"`
	AccumulatedSpecialDeduction  float64                `json:"accumulated_special_deduction"`
	AccumulatedSpecialAdditional float64                `json:"accumulated_special_additional"`
	AccumulatedOtherDeduction    float64                `json:"accumulated_other_deduction"`
	AccumulatedPrepaidTax        float64                `json:"accumulated_prepaid_tax"`
	SpecialAdditionalItems       *SpecialAdditionalItems `json:"special_additional_items,omitempty"`
}

// SpecialAdditionalItems holds the monthly breakdown of 专项附加扣除.
type SpecialAdditionalItems struct {
	ChildrenEducation   int  `json:"children_education"`
	ContinuingEducation bool `json:"continuing_education"`
	HousingLoanInterest bool `json:"housing_loan_interest"`
	HousingRentTier     int  `json:"housing_rent_tier"`
	ElderlySupport      int  `json:"elderly_support"`
	InfantCare          int  `json:"infant_care"`
}

// PITOutput holds the calculated PIT for the current month.
type PITOutput struct {
	MonthlySalary                float64 `json:"monthly_salary"`
	AccumulatedSalary            float64 `json:"accumulated_salary"`
	Months                       int     `json:"months"`
	BasicDeduction               float64 `json:"basic_deduction"`
	AccumulatedSpecialDeduction  float64 `json:"accumulated_special_deduction"`
	AccumulatedSpecialAdditional float64 `json:"accumulated_special_additional"`
	AccumulatedOtherDeduction    float64 `json:"accumulated_other_deduction"`
	TotalDeductions              float64 `json:"total_deductions"`
	AccumulatedTaxableIncome     float64 `json:"accumulated_taxable_income"`
	ApplicableRate               float64 `json:"applicable_rate"`
	QuickDeduction               float64 `json:"quick_deduction"`
	AccumulatedTaxPayable        float64 `json:"accumulated_tax_payable"`
	AccumulatedPrepaidTax        float64 `json:"accumulated_prepaid_tax"`
	CurrentMonthTax              float64 `json:"current_month_tax"`
	Formula                      string  `json:"formula"`
}

// CalcPIT computes monthly individual income tax using the cumulative withholding method.
func CalcPIT(input PITInput) (PITOutput, error) {
	input.MonthlySalary = round2(input.MonthlySalary)
	input.AccumulatedSalary = round2(input.AccumulatedSalary)
	input.AccumulatedSpecialDeduction = round2(input.AccumulatedSpecialDeduction)
	input.AccumulatedSpecialAdditional = round2(input.AccumulatedSpecialAdditional)
	input.AccumulatedOtherDeduction = round2(input.AccumulatedOtherDeduction)
	input.AccumulatedPrepaidTax = round2(input.AccumulatedPrepaidTax)

	if input.MonthlySalary < 0 || input.AccumulatedSalary < 0 {
		return PITOutput{}, fmt.Errorf("salary must be non-negative")
	}
	if input.Months < 1 {
		return PITOutput{}, fmt.Errorf("months must be >= 1")
	}
	if input.AccumulatedSpecialDeduction < 0 || input.AccumulatedSpecialAdditional < 0 ||
		input.AccumulatedOtherDeduction < 0 || input.AccumulatedPrepaidTax < 0 {
		return PITOutput{}, fmt.Errorf("deductions and prepaid tax must be non-negative")
	}
	if input.AccumulatedSalary < input.MonthlySalary {
		return PITOutput{}, fmt.Errorf("accumulated salary must be >= monthly salary")
	}

	if input.SpecialAdditionalItems != nil && input.AccumulatedSpecialAdditional == 0 {
		monthly := CalcMonthlySpecialAdditional(*input.SpecialAdditionalItems)
		input.AccumulatedSpecialAdditional = round2(monthly * float64(input.Months))
	}

	basicDeduction := round2(float64(input.Months) * PITBasicDeductionPerMonth)
	totalDeductions := round2(basicDeduction + input.AccumulatedSpecialDeduction +
		input.AccumulatedSpecialAdditional + input.AccumulatedOtherDeduction)

	taxableIncome := round2(input.AccumulatedSalary - totalDeductions)
	if taxableIncome < 0 {
		taxableIncome = 0
	}

	rate, qd := pitBracket(taxableIncome)
	accumulatedTax := round2(taxableIncome*rate - qd)
	if accumulatedTax < 0 {
		accumulatedTax = 0
	}

	currentMonthTax := round2(accumulatedTax - input.AccumulatedPrepaidTax)
	if currentMonthTax < 0 {
		currentMonthTax = 0
	}

	formula := fmt.Sprintf(
		"(%.2f - %.2f - %.2f - %.2f - %.2f) x %.0f%% - %.2f = %.2f",
		input.AccumulatedSalary,
		basicDeduction,
		input.AccumulatedSpecialDeduction,
		input.AccumulatedSpecialAdditional,
		input.AccumulatedOtherDeduction,
		rate*100,
		qd,
		accumulatedTax,
	)

	return PITOutput{
		MonthlySalary:                input.MonthlySalary,
		AccumulatedSalary:            input.AccumulatedSalary,
		Months:                       input.Months,
		BasicDeduction:               basicDeduction,
		AccumulatedSpecialDeduction:  input.AccumulatedSpecialDeduction,
		AccumulatedSpecialAdditional: input.AccumulatedSpecialAdditional,
		AccumulatedOtherDeduction:    input.AccumulatedOtherDeduction,
		TotalDeductions:              totalDeductions,
		AccumulatedTaxableIncome:     taxableIncome,
		ApplicableRate:               rate,
		QuickDeduction:               qd,
		AccumulatedTaxPayable:        accumulatedTax,
		AccumulatedPrepaidTax:        input.AccumulatedPrepaidTax,
		CurrentMonthTax:              currentMonthTax,
		Formula:                      formula,
	}, nil
}

// CalcMonthlySpecialAdditional computes the monthly total for 专项附加扣除 from item selections.
func CalcMonthlySpecialAdditional(items SpecialAdditionalItems) float64 {
	var total float64
	total += float64(items.ChildrenEducation) * PITChildrenEducation
	if items.ContinuingEducation {
		total += PITContinuingEducation
	}
	if items.HousingLoanInterest {
		total += PITHousingLoanInterest
	}
	switch items.HousingRentTier {
	case 1:
		total += PITHousingRentTier1
	case 2:
		total += PITHousingRentTier2
	case 3:
		total += PITHousingRentTier3
	}
	if items.ElderlySupport == 1 {
		total += PITElderlySupportOnlyChild
	} else if items.ElderlySupport > 1 {
		shared := float64(items.ElderlySupport)
		if shared > PITElderlySupportSharedMax {
			shared = PITElderlySupportSharedMax
		}
		total += shared
	}
	total += float64(items.InfantCare) * PITInfantCare
	return round2(total)
}
