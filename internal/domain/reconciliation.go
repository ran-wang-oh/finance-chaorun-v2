package domain

// LogisticsRecord tracks a shipment linked to an invoice.
type LogisticsRecord struct {
	ID           string `json:"id"`
	EntityID     string `json:"entity_id"`
	InvoiceID    string `json:"invoice_id"`
	WaybillNo    string `json:"waybill_no"`
	Carrier      string `json:"carrier"`
	Status       string `json:"status"`
	ShipDate     string `json:"ship_date,omitempty"`
	DeliveryDate string `json:"delivery_date,omitempty"`
	Items        string `json:"items,omitempty"`
	Notes        string `json:"notes,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

// Logistics status constants.
const (
	LogisticsStatusShipped    = "shipped"
	LogisticsStatusInTransit  = "in_transit"
	LogisticsStatusDelivered  = "delivered"
)

// BankTransaction represents a bank statement line.
type BankTransaction struct {
	ID                 string  `json:"id"`
	EntityID           string  `json:"entity_id"`
	BookID             string  `json:"book_id"`
	TransactionDate    string  `json:"transaction_date"`
	CounterpartyName   string  `json:"counterparty_name"`
	CounterpartyAccount string `json:"counterparty_account,omitempty"`
	Amount             float64 `json:"amount"`
	Direction          string  `json:"direction"`
	Summary            string  `json:"summary,omitempty"`
	BankReference      string  `json:"bank_reference,omitempty"`
	MatchedInvoiceID   string  `json:"matched_invoice_id,omitempty"`
	MatchConfidence    float64 `json:"match_confidence,omitempty"`
	CreatedAt          string  `json:"created_at,omitempty"`
	UpdatedAt          string  `json:"updated_at,omitempty"`
}

// Bank direction constants.
const (
	BankDirectionIn  = "in"
	BankDirectionOut = "out"
)

// Match status constants.
const (
	MatchStatusMatched      = "matched"
	MatchStatusUnmatched    = "unmatched"
	MatchStatusNotApplicable = "not_applicable"
)

// ThreeWayRow represents one row in a three-way match result.
type ThreeWayRow struct {
	InvoiceID       string  `json:"invoice_id"`
	InvoiceNo       string  `json:"invoice_no"`
	CounterpartyName string `json:"counterparty_name"`
	AmountWithTax   float64 `json:"amount_with_tax"`
	Direction       string  `json:"direction"`
	LogisticsStatus string  `json:"logistics_status"`
	LogisticsID     string  `json:"logistics_id,omitempty"`
	WaybillNo       string  `json:"waybill_no,omitempty"`
	BankStatus      string  `json:"bank_status"`
	BankTxID        string  `json:"bank_tx_id,omitempty"`
	BankAmount      float64 `json:"bank_amount,omitempty"`
}

// ThreeWaySummary holds the three-way match results for a period.
type ThreeWaySummary struct {
	Period           string        `json:"period"`
	Rows             []ThreeWayRow `json:"rows"`
	TotalCount       int           `json:"total_count"`
	FullMatch        int           `json:"full_match"`
	MissingLogistics int           `json:"missing_logistics"`
	MissingBank      int           `json:"missing_bank"`
}
