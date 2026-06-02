package domain

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

type Invoice struct {
	ID                 string         `json:"id"`
	EntityID           string         `json:"entity_id"`
	BookID             string         `json:"book_id"`
	InvoiceNo          string         `json:"invoice_no"`
	InvoiceType        string         `json:"invoice_type"`
	Direction          string         `json:"direction"`
	IssueDate          string         `json:"issue_date"`
	SellerName         string         `json:"seller_name,omitempty"`
	SellerTaxNo        string         `json:"seller_tax_no,omitempty"`
	BuyerName          string         `json:"buyer_name,omitempty"`
	BuyerTaxNo         string         `json:"buyer_tax_no,omitempty"`
	AmountWithoutTax   float64        `json:"amount_without_tax"`
	TaxAmount          float64        `json:"tax_amount"`
	AmountWithTax      float64        `json:"amount_with_tax"`
	Currency           string         `json:"currency"`
	Status             string         `json:"status"`
	Source             string         `json:"source"`
	InvoiceKind        string         `json:"invoice_kind,omitempty"`
	DigitalInvoiceNo   string         `json:"digital_invoice_no,omitempty"`
	BusinessTag        string         `json:"business_tag,omitempty"`
	VerificationStatus string         `json:"verification_status,omitempty"`
	UsageStatus        string         `json:"usage_status,omitempty"`
	DeductionStatus    string         `json:"deduction_status,omitempty"`
	RedLetterStatus    string         `json:"red_letter_status,omitempty"`
	OriginalInvoiceID  string         `json:"original_invoice_id,omitempty"`
	TaxAccountPayload  string         `json:"tax_account_payload,omitempty"`
	Extraction         []byte         `json:"extraction,omitempty"`
	EvidenceRefs       []byte         `json:"evidence_refs,omitempty"`
	InvoiceLines       []InvoiceLine  `json:"invoice_lines,omitempty"`
	CreatedAt          string         `json:"created_at"`
	UpdatedAt          string         `json:"updated_at"`
}

type InvoiceLine struct {
	ID             string  `json:"id"`
	EntityID       string  `json:"entity_id"`
	InvoiceID      string  `json:"invoice_id"`
	LineNo         int     `json:"line_no"`
	ItemName       string  `json:"item_name"`
	ItemCode       string  `json:"item_code,omitempty"`
	Quantity       float64 `json:"quantity,omitempty"`
	UnitPrice      float64 `json:"unit_price,omitempty"`
	Amount         float64 `json:"amount"`
	TaxRate        float64 `json:"tax_rate"`
	TaxAmount      float64 `json:"tax_amount"`
	GoodsServiceCode string `json:"goods_service_code,omitempty"`
	Metadata       []byte  `json:"metadata,omitempty"`
}

const (
	DirectionInput  = "input"
	DirectionOutput = "output"
)

const (
	StatusDraft         = "draft"
	StatusPendingReview = "pending_review"
	StatusApproved      = "approved"
	StatusRejected      = "rejected"
	StatusPosted        = "posted"
	StatusVoided        = "voided"
	StatusRedLettered   = "red_lettered"
)

const (
	InvoiceKindVATSpecial   = "vat_special"
	InvoiceKindVATNormal    = "vat_normal"
	InvoiceKindElectronic   = "electronic"
	InvoiceKindFullyDigital = "fully_digital"
)

const (
	VerStatusUnchecked          = "unchecked"
	VerStatusVerified           = "verified"
	VerStatusVerificationFailed = "verification_failed"
)

const (
	UsageStatusUnconfirmed = "unconfirmed"
	UsageStatusConfirmed   = "confirmed"
	UsageStatusPartial     = "partial"
)

const (
	DeductionStatusNotDeducted    = "not_deducted"
	DeductionStatusDeducted       = "deducted"
	DeductionStatusTransferredOut = "transferred_out"
)

const (
	RedLetterStatusNone      = "none"
	RedLetterStatusPartially = "partially"
	RedLetterStatusFully     = "fully"
)

var (
	taxIDPattern = regexp.MustCompile(`^[0-9A-Z]{15}([0-9A-Z]{3})?$`)
	invCodePat   = regexp.MustCompile(`^\d{10}(\d{2})?$`)
	invNoPat     = regexp.MustCompile(`^[A-Za-z0-9-]{1,40}$`)
	validRates   = map[float64]bool{0: true, 0.005: true, 0.01: true, 0.03: true, 0.05: true, 0.06: true, 0.09: true, 0.13: true}
)

type ExtractionResult struct {
	InvoiceType          string   `json:"invoice_type"`
	InvoiceCode          string   `json:"invoice_code,omitempty"`
	InvoiceNo            string   `json:"invoice_no"`
	IssueDate            string   `json:"issue_date"`
	CounterpartyName     string   `json:"counterparty_name"`
	SellerTaxID          string   `json:"seller_tax_id,omitempty"`
	BuyerTaxID           string   `json:"buyer_tax_id,omitempty"`
	TaxCategoryCode      string   `json:"tax_category_code,omitempty"`
	TaxRate              float64  `json:"tax_rate,omitempty"`
	AmountWithoutTax     float64  `json:"amount_without_tax"`
	TaxAmount            float64  `json:"tax_amount"`
	AmountWithTax        float64  `json:"amount_with_tax"`
	Direction            string   `json:"direction,omitempty"`
	DigitalInvoiceNo     string   `json:"digital_invoice_no,omitempty"`
	InvoiceKind          string   `json:"invoice_kind,omitempty"`
	BusinessTag          string   `json:"business_tag,omitempty"`
	SuggestedEntry       string   `json:"suggested_entry"`
	ExtractionConfidence float64  `json:"extraction_confidence"`
	InvoiceLines         []InvoiceLine `json:"invoice_lines,omitempty"`
}

func (e *ExtractionResult) Validate() error {
	if strings.TrimSpace(e.InvoiceNo) == "" {
		return fmt.Errorf("invoice_no is required")
	}
	if !invNoPat.MatchString(strings.TrimSpace(e.InvoiceNo)) {
		return fmt.Errorf("invoice_no format invalid: %s", e.InvoiceNo)
	}
	if e.InvoiceCode != "" && !invCodePat.MatchString(strings.TrimSpace(e.InvoiceCode)) {
		return fmt.Errorf("invoice_code format invalid: %s", e.InvoiceCode)
	}
	if e.SellerTaxID != "" && !taxIDPattern.MatchString(strings.TrimSpace(e.SellerTaxID)) {
		return fmt.Errorf("seller_tax_id format invalid: %s", e.SellerTaxID)
	}
	if e.BuyerTaxID != "" && !taxIDPattern.MatchString(strings.TrimSpace(e.BuyerTaxID)) {
		return fmt.Errorf("buyer_tax_id format invalid: %s", e.BuyerTaxID)
	}
	if e.TaxRate != 0 && !validRates[e.TaxRate] {
		return fmt.Errorf("tax_rate invalid: %.2f", e.TaxRate)
	}
	if e.Direction != "" && e.Direction != DirectionInput && e.Direction != DirectionOutput {
		return fmt.Errorf("direction must be 'input' or 'output': %s", e.Direction)
	}
	diff := math.Abs(e.AmountWithTax - e.AmountWithoutTax - e.TaxAmount)
	if diff > 0.02 {
		return fmt.Errorf("amount mismatch: without_tax + tax_amount != with_tax (diff=%.2f)", diff)
	}
	return nil
}
