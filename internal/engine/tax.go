package engine

import (
	"fmt"
	"math"
)

// TaxpayerType classifies the VAT taxpayer.
type TaxpayerType string

const (
	TaxpayerGeneral    TaxpayerType = "general"
	TaxpayerSmallScale TaxpayerType = "small_scale"
)

// Location for urban construction tax rate determination.
type Location string

const (
	LocationCity  Location = "city"
	LocationTown  Location = "town"
	LocationOther Location = "other"
)

// VATInput holds the data needed for VAT calculation.
type VATInput struct {
	TaxpayerType TaxpayerType `json:"taxpayer_type"`
	OutputTax    float64      `json:"output_tax"`
	InputTax     float64      `json:"input_tax"`
	SalesAmount  float64      `json:"sales_amount"`
	LevyRate     float64      `json:"levy_rate"`
}

// VATOutput holds the calculated VAT and surcharges.
type VATOutput struct {
	OutputTax               float64 `json:"output_tax"`
	InputTax                float64 `json:"input_tax"`
	VATPayable              float64 `json:"vat_payable"`
	UrbanConstructionTax    float64 `json:"urban_construction_tax"`
	EducationSurcharge      float64 `json:"education_surcharge"`
	LocalEducationSurcharge float64 `json:"local_education_surcharge"`
	TotalTax                float64 `json:"total_tax"`
}

// Calculate computes VAT payable and surcharges.
func Calculate(input VATInput) (VATOutput, error) {
	input.OutputTax = round2(input.OutputTax)
	input.InputTax = round2(input.InputTax)
	input.SalesAmount = round2(input.SalesAmount)

	if input.OutputTax < 0 || input.InputTax < 0 || input.SalesAmount < 0 {
		return VATOutput{}, fmt.Errorf("tax amounts must be non-negative")
	}

	var vatPayable float64
	switch input.TaxpayerType {
	case TaxpayerGeneral:
		vatPayable = round2(input.OutputTax - input.InputTax)
	case TaxpayerSmallScale:
		rate := input.LevyRate
		if rate <= 0 {
			rate = 0.03
		}
		vatPayable = round2(input.SalesAmount * rate)
	default:
		return VATOutput{}, fmt.Errorf("unknown taxpayer type: %s", input.TaxpayerType)
	}

	if vatPayable < 0 {
		vatPayable = 0
	}

	out := VATOutput{
		OutputTax:  input.OutputTax,
		InputTax:   input.InputTax,
		VATPayable: vatPayable,
	}
	out.UrbanConstructionTax = round2(vatPayable * urbanConstructionRate(LocationCity))
	out.EducationSurcharge = round2(vatPayable * 0.03)
	out.LocalEducationSurcharge = round2(vatPayable * 0.02)
	out.TotalTax = round2(vatPayable + out.UrbanConstructionTax + out.EducationSurcharge + out.LocalEducationSurcharge)
	return out, nil
}

// CalculateWithLocation computes VAT payable and surcharges with location-specific urban construction tax rate.
func CalculateWithLocation(input VATInput, loc Location) (VATOutput, error) {
	out, err := Calculate(input)
	if err != nil {
		return out, err
	}
	out.UrbanConstructionTax = round2(out.VATPayable * urbanConstructionRate(loc))
	out.TotalTax = round2(out.VATPayable + out.UrbanConstructionTax + out.EducationSurcharge + out.LocalEducationSurcharge)
	return out, nil
}

// CalcSmallScaleVAT computes small-scale taxpayer VAT with exemption.
// Monthly sales <= 100,000 -> exempt. periodMonths typically 3 (quarterly) or 1 (monthly).
func CalcSmallScaleVAT(sales float64, levyRate float64, periodMonths int) (float64, bool) {
	sales = round2(sales)
	if periodMonths <= 0 {
		periodMonths = 3
	}
	if levyRate <= 0 {
		levyRate = 0.03
	}
	monthlyAvg := sales / float64(periodMonths)
	if monthlyAvg <= 100000 {
		return 0, true
	}
	return round2(sales * levyRate), false
}

func urbanConstructionRate(loc Location) float64 {
	switch loc {
	case LocationCity:
		return 0.07
	case LocationTown:
		return 0.05
	default:
		return 0.01
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
