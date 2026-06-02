package capability

type SideEffect string

const (
	SideEffectRead           SideEffect = "read"
	SideEffectDraftWrite     SideEffect = "draft_write"
	SideEffectCommittedWrite SideEffect = "committed_write"
	SideEffectDestructive    SideEffect = "destructive"
)

type Capability struct {
	Name                   string      `json:"name"`
	Domain                 string      `json:"domain"`
	Version                string      `json:"version"`
	SideEffect             SideEffect  `json:"side_effect"`
	RequiresIdempotencyKey bool        `json:"requires_idempotency_key"`
	SupportsDryRun         bool        `json:"supports_dry_run"`
	InputSchema            any         `json:"input_schema"`
	OutputSchema           any         `json:"output_schema"`
}

func Catalog() []Capability {
	return []Capability{
		{
			Name:                   "finance.book.create",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectCommittedWrite,
			RequiresIdempotencyKey: true,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"name", "currency"},
				"properties": map[string]any{
					"name":                 map[string]any{"type": "string"},
					"currency":             map[string]any{"type": "string"},
					"accounting_standard":   map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":   map[string]any{"type": "string"},
					"name": map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:                   "finance.book.list",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema:  map[string]any{"type": "object", "properties": map[string]any{}},
			OutputSchema: map[string]any{"type": "object", "properties": map[string]any{"books": map[string]any{"type": "array"}}},
		},
		{
			Name:                   "finance.account.list",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id"},
				"properties": map[string]any{
					"book_id":  map[string]any{"type": "string"},
					"category": map[string]any{"type": "string"},
					"limit":    map[string]any{"type": "integer"},
				},
			},
			OutputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"items": map[string]any{"type": "array"}},
			},
		},
		{
			Name:                   "finance.invoice.create_draft",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectDraftWrite,
			RequiresIdempotencyKey: true,
			SupportsDryRun:         true,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "invoice_no", "direction", "issue_date", "amount_without_tax", "tax_amount", "amount_with_tax"},
				"properties": map[string]any{
					"book_id":            map[string]any{"type": "string"},
					"invoice_no":         map[string]any{"type": "string"},
					"invoice_type":       map[string]any{"type": "string"},
					"direction":          map[string]any{"type": "string", "enum": []string{"input", "output"}},
					"issue_date":         map[string]any{"type": "string"},
					"seller_name":        map[string]any{"type": "string"},
					"seller_tax_no":      map[string]any{"type": "string"},
					"buyer_name":         map[string]any{"type": "string"},
					"buyer_tax_no":       map[string]any{"type": "string"},
					"amount_without_tax": map[string]any{"type": "number"},
					"tax_amount":         map[string]any{"type": "number"},
					"amount_with_tax":    map[string]any{"type": "number"},
					"currency":           map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"invoice_id": map[string]any{"type": "string"},
					"status":     map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:                   "finance.invoice.approve",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectCommittedWrite,
			RequiresIdempotencyKey: true,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"invoice_id"},
				"properties": map[string]any{
					"invoice_id": map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"invoice_id":      map[string]any{"type": "string"},
					"journal_entry_id": map[string]any{"type": "string"},
					"status":           map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:                   "finance.invoice.reject",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectCommittedWrite,
			RequiresIdempotencyKey: true,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"invoice_id"},
				"properties": map[string]any{
					"invoice_id": map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"invoice_id": map[string]any{"type": "string"},
					"status":     map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:                   "finance.journal.create_draft",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectDraftWrite,
			RequiresIdempotencyKey: true,
			SupportsDryRun:         true,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "period", "summary", "lines"},
				"properties": map[string]any{
					"book_id":  map[string]any{"type": "string"},
					"period":   map[string]any{"type": "string"},
					"summary":  map[string]any{"type": "string"},
					"lines": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"required": []string{"account_code", "direction", "debit_amount", "credit_amount"},
							"properties": map[string]any{
								"account_code": map[string]any{"type": "string"},
								"account_name": map[string]any{"type": "string"},
								"direction":    map[string]any{"type": "string", "enum": []string{"debit", "credit"}},
								"debit_amount": map[string]any{"type": "number"},
								"credit_amount": map[string]any{"type": "number"},
							},
						},
					},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"journal_entry_id": map[string]any{"type": "string"},
					"voucher_no":       map[string]any{"type": "string"},
					"status":           map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:                   "finance.journal.post",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectCommittedWrite,
			RequiresIdempotencyKey: true,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"journal_entry_id"},
				"properties": map[string]any{
					"journal_entry_id": map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"journal_entry_id": map[string]any{"type": "string"},
					"status":           map[string]any{"type": "string"},
					"voucher_no":       map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:                   "finance.journal.void",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectDestructive,
			RequiresIdempotencyKey: true,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"journal_entry_id"},
				"properties": map[string]any{
					"journal_entry_id": map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"journal_entry_id": map[string]any{"type": "string"},
					"status":           map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:                   "finance.report.trial_balance",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "period"},
				"properties": map[string]any{
					"book_id": map[string]any{"type": "string"},
					"period":  map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"period":       map[string]any{"type": "string"},
					"total_debit":  map[string]any{"type": "number"},
					"total_credit": map[string]any{"type": "number"},
					"balanced":     map[string]any{"type": "boolean"},
				},
			},
		},
		{
			Name:                   "finance.report.account_balance",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "period"},
				"properties": map[string]any{
					"book_id": map[string]any{"type": "string"},
					"period":  map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"period": map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:                   "finance.period.close_check",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "period"},
				"properties": map[string]any{
					"book_id": map[string]any{"type": "string"},
					"period":  map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"passed":  map[string]any{"type": "boolean"},
					"period":  map[string]any{"type": "string"},
					"summary": map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:                   "finance.period.close",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectCommittedWrite,
			RequiresIdempotencyKey: true,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "period"},
				"properties": map[string]any{
					"book_id": map[string]any{"type": "string"},
					"period":  map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"period": map[string]any{"type": "string"},
					"status": map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:                   "finance.period.lock",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectCommittedWrite,
			RequiresIdempotencyKey: true,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "period"},
				"properties": map[string]any{
					"book_id": map[string]any{"type": "string"},
					"period":  map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"period": map[string]any{"type": "string"},
					"status": map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:                   "finance.period.reopen",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectDestructive,
			RequiresIdempotencyKey: true,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "period"},
				"properties": map[string]any{
					"book_id": map[string]any{"type": "string"},
					"period":  map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"period": map[string]any{"type": "string"},
					"status": map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:                   "finance.reconciliation.upsert_logistics",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectCommittedWrite,
			RequiresIdempotencyKey: true,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"invoice_id", "waybill_no"},
				"properties": map[string]any{
					"invoice_id": map[string]any{"type": "string"},
					"waybill_no": map[string]any{"type": "string"},
					"carrier":    map[string]any{"type": "string"},
					"status":     map[string]any{"type": "string", "enum": []string{"shipped", "in_transit", "delivered"}},
					"ship_date":  map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"logistics_id": map[string]any{"type": "string"}},
			},
		},
		{
			Name:                   "finance.reconciliation.upsert_bank_transaction",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectCommittedWrite,
			RequiresIdempotencyKey: true,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"transaction_date", "counterparty_name", "amount", "direction"},
				"properties": map[string]any{
					"transaction_date":    map[string]any{"type": "string"},
					"counterparty_name":   map[string]any{"type": "string"},
					"counterparty_account": map[string]any{"type": "string"},
					"amount":              map[string]any{"type": "number"},
					"direction":           map[string]any{"type": "string", "enum": []string{"in", "out"}},
					"summary":             map[string]any{"type": "string"},
					"bank_reference":      map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"bank_transaction_id": map[string]any{"type": "string"}},
			},
		},
		{
			Name:                   "finance.reconciliation.match_bank",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectCommittedWrite,
			RequiresIdempotencyKey: true,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"bank_transaction_id", "invoice_id"},
				"properties": map[string]any{
					"bank_transaction_id": map[string]any{"type": "string"},
					"invoice_id":          map[string]any{"type": "string"},
					"confidence":          map[string]any{"type": "number"},
				},
			},
			OutputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"status": map[string]any{"type": "string"}},
			},
		},
		{
			Name:                   "finance.reconciliation.unmatch_bank",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectCommittedWrite,
			RequiresIdempotencyKey: true,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"bank_transaction_id"},
				"properties": map[string]any{
					"bank_transaction_id": map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"status": map[string]any{"type": "string"}},
			},
		},
		{
			Name:                   "finance.report.profit_statement",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "period"},
				"properties": map[string]any{
					"book_id":             map[string]any{"type": "string"},
					"period":              map[string]any{"type": "string"},
					"accounting_standard": map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"revenue":          map[string]any{"type": "number"},
					"operating_profit": map[string]any{"type": "number"},
					"net_profit":       map[string]any{"type": "number"},
				},
			},
		},
		{
			Name:                   "finance.report.balance_sheet",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "period"},
				"properties": map[string]any{
					"book_id":             map[string]any{"type": "string"},
					"period":              map[string]any{"type": "string"},
					"accounting_standard": map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"total_assets":      map[string]any{"type": "number"},
					"total_liabilities": map[string]any{"type": "number"},
					"total_equity":       map[string]any{"type": "number"},
				},
			},
		},
		{
			Name:                   "finance.report.vat_cross_check",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "period"},
				"properties": map[string]any{
					"book_id": map[string]any{"type": "string"},
					"period":  map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"output_total": map[string]any{"type": "number"},
					"input_total":  map[string]any{"type": "number"},
					"net_payable":  map[string]any{"type": "number"},
				},
			},
		},
		{
			Name:                   "finance.report.vat_return",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "period"},
				"properties": map[string]any{
					"book_id": map[string]any{"type": "string"},
					"period":  map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"main": map[string]any{"type": "object"},
				},
			},
		},
		{
			Name:                   "finance.report.cross_tax_validation",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "period"},
				"properties": map[string]any{
					"book_id": map[string]any{"type": "string"},
					"period":  map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"vat_sales":   map[string]any{"type": "number"},
					"cit_revenue":  map[string]any{"type": "number"},
					"consistent":  map[string]any{"type": "boolean"},
				},
			},
		},
		{
			Name:                   "finance.report.three_way_match",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "period"},
				"properties": map[string]any{
					"book_id": map[string]any{"type": "string"},
					"period":  map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"period":      map[string]any{"type": "string"},
					"total_count":  map[string]any{"type": "integer"},
					"full_match":   map[string]any{"type": "integer"},
					"missing_bank": map[string]any{"type": "integer"},
				},
			},
		},
		{
			Name:                   "finance.tax.calculate_vat",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"taxpayer_type", "output_tax", "input_tax"},
				"properties": map[string]any{
					"taxpayer_type": map[string]any{"type": "string", "enum": []string{"general", "small_scale"}},
					"output_tax":    map[string]any{"type": "number"},
					"input_tax":     map[string]any{"type": "number"},
					"sales_amount":  map[string]any{"type": "number"},
					"levy_rate":     map[string]any{"type": "number"},
					"location":      map[string]any{"type": "string", "enum": []string{"city", "town", "other"}},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"vat_payable": map[string]any{"type": "number"},
					"total_tax":   map[string]any{"type": "number"},
				},
			},
		},
		{
			Name:                   "finance.tax.calculate_stamp",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"category", "amount"},
				"properties": map[string]any{
					"category": map[string]any{"type": "string"},
					"amount":   map[string]any{"type": "number"},
					"count":    map[string]any{"type": "integer"},
				},
			},
			OutputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"tax_payable": map[string]any{"type": "number"}},
			},
		},
		{
			Name:                   "finance.tax.calculate_pit",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"monthly_salary", "accumulated_salary", "months"},
				"properties": map[string]any{
					"monthly_salary": map[string]any{"type": "number"},
					"accumulated_salary": map[string]any{"type": "number"},
					"months": map[string]any{"type": "integer"},
				},
			},
			OutputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"current_month_tax": map[string]any{"type": "number"}},
			},
		},
		{
			Name:                   "finance.report.cit_return",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "tax_year"},
				"properties": map[string]any{
					"book_id":  map[string]any{"type": "string"},
					"tax_year": map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"taxable_income": map[string]any{"type": "number"},
					"tax_payable":    map[string]any{"type": "number"},
					"tax_due":        map[string]any{"type": "number"},
				},
			},
		},
		{
			Name:                   "finance.tax.list_adjustments",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"tax_year"},
				"properties": map[string]any{
					"tax_year": map[string]any{"type": "string"},
					"category": map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"items": map[string]any{"type": "array"}},
			},
		},
		{
			Name:                   "finance.tax.upsert_adjustments",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectCommittedWrite,
			RequiresIdempotencyKey: true,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "tax_year", "adjustments"},
				"properties": map[string]any{
					"book_id":     map[string]any{"type": "string"},
					"tax_year":    map[string]any{"type": "string"},
					"adjustments": map[string]any{"type": "array"},
				},
			},
			OutputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"status": map[string]any{"type": "string"}},
			},
		},
		{
			Name:                   "finance.report.tax_risk",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "period"},
				"properties": map[string]any{
					"book_id": map[string]any{"type": "string"},
					"period":  map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"total_risks":  map[string]any{"type": "integer"},
					"danger_count": map[string]any{"type": "integer"},
				},
			},
		},
		{
			Name:                   "finance.risk.scan",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "period"},
				"properties": map[string]any{
					"book_id": map[string]any{"type": "string"},
					"period":  map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"findings": map[string]any{"type": "array"}},
			},
		},
		{
			Name:                   "finance.consistency.check",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "period"},
				"properties": map[string]any{
					"book_id": map[string]any{"type": "string"},
					"period":  map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"checks": map[string]any{"type": "array"}},
			},
		},
		{
			Name:                   "finance.period.enhanced_close_check",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "period"},
				"properties": map[string]any{
					"book_id": map[string]any{"type": "string"},
					"period":  map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"passed":  map[string]any{"type": "boolean"},
					"summary": map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:                   "finance.invoice.create_red_letter",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectDraftWrite,
			RequiresIdempotencyKey: true,
			SupportsDryRun:         true,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"original_invoice_id", "red_type"},
				"properties": map[string]any{
					"book_id":              map[string]any{"type": "string"},
					"original_invoice_id":  map[string]any{"type": "string"},
					"red_type":             map[string]any{"type": "string", "enum": []string{"partially", "fully"}},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"invoice_id": map[string]any{"type": "string"},
					"status":     map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:                   "finance.invoice.import_einvoice",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectDraftWrite,
			RequiresIdempotencyKey: true,
			SupportsDryRun:         true,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"einvoice"},
				"properties": map[string]any{
					"book_id":  map[string]any{"type": "string"},
					"einvoice": map[string]any{"type": "object"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"invoice_id": map[string]any{"type": "string"},
					"status":     map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:                   "finance.invoice.verify",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectCommittedWrite,
			RequiresIdempotencyKey: true,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"invoice_id"},
				"properties": map[string]any{
					"invoice_id": map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"status": map[string]any{"type": "string"}},
			},
		},
		{
			Name:                   "finance.invoice.confirm_usage",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectCommittedWrite,
			RequiresIdempotencyKey: true,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"invoice_id", "usage_status"},
				"properties": map[string]any{
					"invoice_id":   map[string]any{"type": "string"},
					"usage_status": map[string]any{"type": "string", "enum": []string{"confirmed", "partial", "unconfirmed"}},
				},
			},
			OutputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"status": map[string]any{"type": "string"}},
			},
		},
		{
			Name:                   "finance.journal.update_draft",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectDraftWrite,
			RequiresIdempotencyKey: true,
			SupportsDryRun:         true,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"journal_entry_id", "summary", "lines"},
				"properties": map[string]any{
					"journal_entry_id": map[string]any{"type": "string"},
					"summary":          map[string]any{"type": "string"},
					"lines":            map[string]any{"type": "array"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"journal_entry_id": map[string]any{"type": "string"},
					"status":           map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:                   "finance.journal.batch_post",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectCommittedWrite,
			RequiresIdempotencyKey: true,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"journal_entry_ids"},
				"properties": map[string]any{
					"journal_entry_ids": map[string]any{
						"type": "array",
						"items": map[string]any{"type": "string"},
					},
				},
			},
			OutputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"count": map[string]any{"type": "integer"}},
			},
		},
		{
			Name:                   "finance.export.trial_balance",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "period"},
				"properties": map[string]any{
					"book_id": map[string]any{"type": "string"},
					"period":  map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"csv": map[string]any{"type": "string"}},
			},
		},
		{
			Name:                   "finance.export.vat_summary",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "period"},
				"properties": map[string]any{
					"book_id": map[string]any{"type": "string"},
					"period":  map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"csv": map[string]any{"type": "string"}},
			},
		},
		{
			Name:                   "finance.export.vat_return",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "period"},
				"properties": map[string]any{
					"book_id": map[string]any{"type": "string"},
					"period":  map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"data": map[string]any{"type": "object"}},
			},
		},
		{
			Name:                   "finance.export.cit_return",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"book_id", "tax_year"},
				"properties": map[string]any{
					"book_id":  map[string]any{"type": "string"},
					"tax_year": map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"data": map[string]any{"type": "object"}},
			},
		},
		{
			Name:                   "finance.export.adjustments",
			Domain:                 "finance",
			Version:                "0.1.0",
			SideEffect:             SideEffectRead,
			RequiresIdempotencyKey: false,
			SupportsDryRun:         false,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"tax_year"},
				"properties": map[string]any{
					"tax_year": map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"data": map[string]any{"type": "array"}},
			},
		},
	}
}
