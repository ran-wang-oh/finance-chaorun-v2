package engine

import "fmt"

// AdjustmentResult holds the result of a single tax adjustment calculation.
type AdjustmentResult struct {
	Category   string  `json:"category"`
	BookAmount float64 `json:"book_amount"`
	Deductible float64 `json:"deductible"`
	Adjustment float64 `json:"adjustment"`
	Formula    string  `json:"formula"`
}

// CalcEntertainmentAdjustment computes the business entertainment expense adjustment.
// deductible = min(60% x actual, 0.5% x sales_revenue).
func CalcEntertainmentAdjustment(actualExpense, salesRevenue float64) (AdjustmentResult, error) {
	actualExpense = round2(actualExpense)
	salesRevenue = round2(salesRevenue)
	if actualExpense < 0 || salesRevenue < 0 {
		return AdjustmentResult{}, fmt.Errorf("amounts must be non-negative")
	}
	limitByExpense := round2(actualExpense * 0.60)
	limitByRevenue := round2(salesRevenue * 0.005)
	deductible := limitByExpense
	if limitByRevenue < deductible {
		deductible = limitByRevenue
	}
	adjustment := round2(actualExpense - deductible)
	return AdjustmentResult{
		Category:   "业务招待费",
		BookAmount: actualExpense,
		Deductible: deductible,
		Adjustment: adjustment,
		Formula:    fmt.Sprintf("min(%.2fx60%%, %.2fx0.5%%) = %.2f", actualExpense, salesRevenue, deductible),
	}, nil
}

// CalcAdvertisingAdjustment computes the advertising expense adjustment.
// deductible = min(actual, 15% x sales_revenue).
func CalcAdvertisingAdjustment(actualExpense, salesRevenue float64) (AdjustmentResult, error) {
	actualExpense = round2(actualExpense)
	salesRevenue = round2(salesRevenue)
	if actualExpense < 0 || salesRevenue < 0 {
		return AdjustmentResult{}, fmt.Errorf("amounts must be non-negative")
	}
	limit := round2(salesRevenue * 0.15)
	deductible := limit
	if actualExpense < deductible {
		deductible = actualExpense
	}
	adjustment := round2(actualExpense - deductible)
	return AdjustmentResult{
		Category:   "广告费",
		BookAmount: actualExpense,
		Deductible: deductible,
		Adjustment: adjustment,
		Formula:    fmt.Sprintf("min(%.2f, %.2fx15%%) = %.2f", actualExpense, salesRevenue, deductible),
	}, nil
}

// CalcWelfareAdjustment computes the employee welfare expense adjustment.
// deductible = min(actual, 14% x total_salary).
func CalcWelfareAdjustment(actualExpense, totalSalary float64) (AdjustmentResult, error) {
	actualExpense = round2(actualExpense)
	totalSalary = round2(totalSalary)
	if actualExpense < 0 || totalSalary < 0 {
		return AdjustmentResult{}, fmt.Errorf("amounts must be non-negative")
	}
	limit := round2(totalSalary * 0.14)
	deductible := limit
	if actualExpense < deductible {
		deductible = actualExpense
	}
	adjustment := round2(actualExpense - deductible)
	return AdjustmentResult{
		Category:   "职工福利费",
		BookAmount: actualExpense,
		Deductible: deductible,
		Adjustment: adjustment,
		Formula:    fmt.Sprintf("min(%.2f, %.2fx14%%) = %.2f", actualExpense, totalSalary, deductible),
	}, nil
}

// CalcRDSuperDeduction computes the R&D super-deduction.
func CalcRDSuperDeduction(rdExpense, rate float64) (AdjustmentResult, error) {
	rdExpense = round2(rdExpense)
	if rdExpense < 0 {
		return AdjustmentResult{}, fmt.Errorf("rd expense must be non-negative")
	}
	if rate <= 0 {
		rate = 1.0
	}
	deduction := round2(rdExpense * rate)
	return AdjustmentResult{
		Category:   "研发费用加计扣除",
		BookAmount: rdExpense,
		Deductible: rdExpense,
		Adjustment: round2(-deduction),
		Formula:    fmt.Sprintf("%.2f x %.0f%% = %.2f (调减)", rdExpense, rate*100, deduction),
	}, nil
}

// CalcAssetImpairmentAdjustment computes the asset impairment adjustment (full add-back).
func CalcAssetImpairmentAdjustment(impairment float64) (AdjustmentResult, error) {
	impairment = round2(impairment)
	if impairment < 0 {
		return AdjustmentResult{}, fmt.Errorf("impairment must be non-negative")
	}
	return AdjustmentResult{
		Category:   "资产减值损失",
		BookAmount: impairment,
		Deductible: 0,
		Adjustment: impairment,
		Formula:    fmt.Sprintf("%.2f 全额调增 (资产减值不得税前扣除)", impairment),
	}, nil
}
