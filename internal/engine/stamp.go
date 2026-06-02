package engine

import "fmt"

// Stamp tax rates (印花税法 2022).
const (
	StampRateSalesContract    = 0.0003
	StampRateLoanContract     = 0.00005
	StampRateLeaseContract    = 0.001
	StampRateTechContract     = 0.0003
	StampRatePropertyContract = 0.0005
	StampBookUnitTax          = 5.0
)

// StampTaxInput holds the data for stamp tax calculation.
type StampTaxInput struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
	Count    int     `json:"count"`
}

// StampTaxOutput holds calculated stamp tax.
type StampTaxOutput struct {
	Category   string  `json:"category"`
	Amount     float64 `json:"amount"`
	Rate       float64 `json:"rate"`
	UnitTax    float64 `json:"unit_tax"`
	TaxPayable float64 `json:"tax_payable"`
}

// CalcStampTax computes stamp tax for a given category.
func CalcStampTax(input StampTaxInput) (StampTaxOutput, error) {
	input.Amount = round2(input.Amount)
	if input.Amount < 0 {
		return StampTaxOutput{}, fmt.Errorf("amount must be non-negative")
	}
	if input.Count < 0 {
		return StampTaxOutput{}, fmt.Errorf("count must be non-negative")
	}

	out := StampTaxOutput{Category: input.Category, Amount: input.Amount}

	switch input.Category {
	case "购销合同":
		out.Rate = StampRateSalesContract
		out.TaxPayable = round2(input.Amount * out.Rate)
	case "借款合同":
		out.Rate = StampRateLoanContract
		out.TaxPayable = round2(input.Amount * out.Rate)
	case "租赁合同":
		out.Rate = StampRateLeaseContract
		out.TaxPayable = round2(input.Amount * out.Rate)
	case "技术合同":
		out.Rate = StampRateTechContract
		out.TaxPayable = round2(input.Amount * out.Rate)
	case "产权转移书据":
		out.Rate = StampRatePropertyContract
		out.TaxPayable = round2(input.Amount * out.Rate)
	case "账簿":
		out.UnitTax = StampBookUnitTax
		n := input.Count
		if n <= 0 {
			n = 1
		}
		out.TaxPayable = round2(float64(n) * StampBookUnitTax)
	default:
		out.Rate = StampRateSalesContract
		out.TaxPayable = round2(input.Amount * out.Rate)
	}
	return out, nil
}
