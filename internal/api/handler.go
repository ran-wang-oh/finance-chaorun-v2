package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"finance.chao.run/v2/internal/capability"
	"finance.chao.run/v2/internal/domain"
	"finance.chao.run/v2/internal/engine"
	"finance.chao.run/v2/internal/provider"
	"finance.chao.run/v2/internal/service"
	"finance.chao.run/v2/internal/store"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc          *service.Service
	capabilities []capability.Capability
	idempStore   IdempotencyStore
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{
		svc:          svc,
		capabilities: capability.Catalog(),
	}
}

func (h *Handler) WithIdempotencyStore(s IdempotencyStore) *Handler {
	h.idempStore = s
	return h
}

func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *Handler) ListCapabilities(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"capabilities": h.capabilities})
}

func (h *Handler) GetContext(w http.ResponseWriter, r *http.Request) {
	var req provider.ProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "", "invalid_request", "malformed body")
		return
	}

	traceID := req.TraceID
	if req.TraceID == "" {
		writeError(w, http.StatusBadRequest, "", "invalid_request", "trace_id is required")
		return
	}

	if req.EntityID == "" {
		writeError(w, http.StatusBadRequest, traceID, "invalid_request", "entity_id is required")
		return
	}

	entityID := req.EntityID
	bookID := r.URL.Query().Get("book_id")

	if h.svc != nil {
		if bookID != "" {
			if resolved, err := h.svc.ResolveBook(r.Context(), entityID, bookID); err == nil {
				bookID = resolved
			}
		}
		ctx := r.Context()
		summary := map[string]any{}

		pendingInv := h.getPendingInvoiceCount(ctx, entityID, bookID)
		summary["pending_invoice_count"] = pendingInv

		draftJE := h.getDraftJournalCount(ctx, entityID, bookID)
		summary["draft_journal_count"] = draftJE

		books, _ := h.svc.ListBooks(ctx, entityID)
		summary["book_count"] = len(books)

		if bookID != "" {
			if unmatchedCount, err := h.svc.UnmatchedBankCount(ctx, entityID, bookID); err == nil {
				summary["unmatched_bank_count"] = unmatchedCount
			}
			currentMonth := time.Now().UTC().Format("2006-01")
			if periods, err := h.svc.ListPeriods(ctx, entityID, bookID); err == nil {
				lastClosed := ""
				for _, p := range periods {
					if p.Status == domain.PeriodStatusClosed || p.Status == domain.PeriodStatusLocked {
						if p.Period > lastClosed {
							lastClosed = p.Period
						}
					}
				}
				summary["last_closed_period"] = lastClosed
				summary["current_period"] = currentMonth
			}
		}

		data, _ := json.Marshal(summary)
		writeJSON(w, http.StatusOK, &provider.ProviderResponse{
			Status:            "ok",
			Data:              data,
			ExternalRequestID: uuid.NewString(),
			TraceID:           traceID,
		})
		return
	}

	writeJSON(w, http.StatusOK, &provider.ProviderResponse{
		Status:            "ok",
		ExternalRequestID: uuid.NewString(),
		TraceID:           traceID,
	})
}

func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	capabilityID := chi.URLParam(r, "capability_id")
	cap := h.findCapability(capabilityID)
	if cap == nil {
		writeError(w, http.StatusNotFound, "", "not_found", "capability not found: "+capabilityID)
		return
	}

	if !cap.SupportsDryRun {
		writeError(w, http.StatusBadRequest, "", "invalid_request", "capability does not support preview")
		return
	}

	var req provider.ProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "", "invalid_request", "malformed body")
		return
	}

	if req.TraceID == "" {
		writeError(w, http.StatusBadRequest, "", "invalid_request", "trace_id is required")
		return
	}

	writeJSON(w, http.StatusOK, &provider.ProviderResponse{
		Status:            "ok",
		ExternalRequestID: uuid.NewString(),
		TraceID:           req.TraceID,
		Warnings:          []string{"preview mode: no data will be written"},
	})
}

func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	capabilityID := chi.URLParam(r, "capability_id")
	cap := h.findCapability(capabilityID)
	if cap == nil {
		writeError(w, http.StatusNotFound, "", "not_found", "capability not found: "+capabilityID)
		return
	}

	var req provider.ProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "", "invalid_request", "malformed body")
		return
	}

	if req.TraceID == "" {
		writeError(w, http.StatusBadRequest, "", "invalid_request", "trace_id is required")
		return
	}

	warnings := []string{}

	switch capabilityID {
	case "finance.journal.create_draft":
		var entry domain.JournalEntry
		if err := json.Unmarshal(req.Input, &entry); err == nil {
			var debit, credit float64
			for _, l := range entry.Lines {
				debit += l.DebitAmount
				credit += l.CreditAmount
			}
			if debit != credit {
				warnings = append(warnings, fmt.Sprintf("journal not balanced: debit=%.2f, credit=%.2f", debit, credit))
			}
		}
	case "finance.invoice.create_draft":
		var inv domain.Invoice
		if err := json.Unmarshal(req.Input, &inv); err == nil {
			extraction := domain.ExtractionResult{
				InvoiceNo:        inv.InvoiceNo,
				AmountWithoutTax: inv.AmountWithoutTax,
				TaxAmount:        inv.TaxAmount,
				AmountWithTax:    inv.AmountWithTax,
				Direction:        inv.Direction,
			}
			if err := extraction.Validate(); err != nil {
				warnings = append(warnings, err.Error())
			}
		}
	}

	resp := &provider.ProviderResponse{
		Status:            "ok",
		ExternalRequestID: uuid.NewString(),
		TraceID:           req.TraceID,
		Warnings:          warnings,
	}
	if len(warnings) > 0 {
		resp.Status = "blocked"
	}
	writeJSON(w, http.StatusOK, resp)
}

// ExecuteV3 handles V3 connector requests: POST /v1/capabilities/{id}
// Accepts the V3 envelope {context: {...}, input: ...} and translates to internal format.
func (h *Handler) ExecuteV3(w http.ResponseWriter, r *http.Request) {
	capabilityID := chi.URLParam(r, "capability_id")
	cap := h.findCapability(capabilityID)
	if cap == nil {
		writeError(w, http.StatusNotFound, "", "not_found", "capability not found: "+capabilityID)
		return
	}

	var v3req provider.V3Request
	if err := json.NewDecoder(r.Body).Decode(&v3req); err != nil {
		writeError(w, http.StatusBadRequest, "", "invalid_request", "malformed V3 request body")
		return
	}

	// Fall back to V2 format if context is empty (backward compat)
	if v3req.Context.EntityID == "" {
		h.Execute(w, r)
		return
	}

	req := v3req.ToProviderRequest(capabilityID)
	// Allow X-Idempotency-Key header to override (V3 sends it via header)
	if idemKey := r.Header.Get("X-Idempotency-Key"); idemKey != "" {
		req.IdempotencyKey = idemKey
	}

	if req.TraceID == "" {
		req.TraceID = r.Header.Get("X-Request-Id")
	}
	if req.TraceID == "" {
		req.TraceID = uuid.NewString()
	}

	h.executeRequest(w, r, capabilityID, cap, req)
}

func (h *Handler) Execute(w http.ResponseWriter, r *http.Request) {
	capabilityID := chi.URLParam(r, "capability_id")
	cap := h.findCapability(capabilityID)
	if cap == nil {
		writeError(w, http.StatusNotFound, "", "not_found", "capability not found: "+capabilityID)
		return
	}

	var req provider.ProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "", "invalid_request", "malformed body")
		return
	}

	// Support X-Idempotency-Key header as fallback (V3 HTTPExecutor)
	if req.IdempotencyKey == "" {
		if idemKey := r.Header.Get("X-Idempotency-Key"); idemKey != "" {
			req.IdempotencyKey = idemKey
		}
	}

	traceID := req.TraceID

	if req.TraceID == "" {
		writeError(w, http.StatusBadRequest, "", "invalid_request", "trace_id is required")
		return
	}

	if req.EntityID == "" {
		writeError(w, http.StatusBadRequest, traceID, "invalid_request", "entity_id is required")
		return
	}

	h.executeRequest(w, r, capabilityID, cap, req)
}

// executeRequest is the shared execution path for both V2 and V3 requests.
func (h *Handler) executeRequest(w http.ResponseWriter, r *http.Request, capabilityID string, cap *capability.Capability, req provider.ProviderRequest) {
	traceID := req.TraceID
	if traceID == "" {
		traceID = uuid.NewString()
	}

	// Idempotency check for write operations
	if cap.RequiresIdempotencyKey {
		if req.IdempotencyKey == "" {
			writeError(w, http.StatusBadRequest, traceID, "invalid_request", "idempotency_key is required for this capability")
			return
		}

		if h.idempStore != nil {
			existing, err := h.idempStore.Get(r.Context(), req.EntityID, capabilityID, req.IdempotencyKey)
			if err != nil {
				writeError(w, http.StatusInternalServerError, traceID, "provider_internal", "idempotency lookup failed")
				return
			}
			if existing != nil {
				inputHash := HashInput(req.Input)
				if existing.InputHash != "" && existing.InputHash != inputHash {
					writeError(w, http.StatusConflict, traceID, "finance_idempotency_conflict",
						"idempotency key reused with different input")
					return
				}
				result, _ := json.Marshal(existing.Result)
				resp := &provider.ProviderResponse{
					Status:            existing.Status,
					Data:              result,
					ExternalRequestID: uuid.NewString(),
					TraceID:           traceID,
				}
				writeJSON(w, http.StatusOK, resp)
				return
			}
		}
	}

	// Execute capability
	ctx := r.Context()
	ctx = provider.ContextWithTraceID(ctx, traceID)
	ctx = provider.ContextWithCapabilityID(ctx, capabilityID)
	ctx = provider.ContextWithIdempotencyKey(ctx, req.IdempotencyKey)
	ctx = provider.ContextWithActorID(ctx, req.Actor.ID)
	result, err := h.executeCapability(ctx, req.EntityID, capabilityID, req)
	if err != nil {
		if cap.RequiresIdempotencyKey && h.idempStore != nil {
			h.idempStore.Save(r.Context(), &store.IdempotencyRecord{
				EntityID:       req.EntityID,
				CapabilityID:   capabilityID,
				IdempotencyKey: req.IdempotencyKey,
				InputHash:      HashInput(req.Input),
				Result:         []byte("{}"),
				Status:         "failed",
			})
		}
		writeError(w, http.StatusOK, traceID, "execution_failed", err.Error())
		return
	}

	if cap.RequiresIdempotencyKey && h.idempStore != nil {
		data, _ := json.Marshal(result)
		h.idempStore.Save(r.Context(), &store.IdempotencyRecord{
			EntityID:       req.EntityID,
			CapabilityID:   capabilityID,
			IdempotencyKey: req.IdempotencyKey,
			InputHash:      HashInput(req.Input),
			Result:         data,
			Status:         "completed",
		})
	}

	data, _ := json.Marshal(result)
	resp := &provider.ProviderResponse{
		Status:            "ok",
		Data:              data,
		ExternalRequestID: uuid.NewString(),
		TraceID:           traceID,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetResource(w http.ResponseWriter, r *http.Request) {
	resourceURI := chi.URLParam(r, "resource_uri")
	if resourceURI == "" {
		writeError(w, http.StatusBadRequest, "", "invalid_request", "resource_uri is required")
		return
	}

	entityID := r.URL.Query().Get("entity_id")
	traceID := r.URL.Query().Get("trace_id")

	if entityID == "" {
		writeError(w, http.StatusBadRequest, traceID, "invalid_request", "entity_id is required")
		return
	}
	if traceID == "" {
		writeError(w, http.StatusBadRequest, "", "invalid_request", "trace_id is required")
		return
	}

	parts := strings.SplitN(resourceURI, "://", 2)
	if len(parts) != 2 {
		writeError(w, http.StatusBadRequest, traceID, "invalid_request", "invalid resource uri")
		return
	}
	path := strings.SplitN(parts[1], "/", 2)
	if len(path) < 2 {
		writeError(w, http.StatusBadRequest, traceID, "invalid_request", "invalid resource uri")
		return
	}

	var result any
	var err error

	switch path[0] {
	case "invoice":
		result, err = h.svc.GetInvoice(r.Context(), entityID, path[1])
	case "journal-entry":
		result, err = h.svc.GetJournalEntry(r.Context(), entityID, path[1])
	case "logistics":
		result, err = h.svc.GetLogisticsByInvoice(r.Context(), entityID, path[1])
	case "bank-transaction":
		result, err = h.svc.GetBankTransaction(r.Context(), entityID, path[1])
	default:
		writeError(w, http.StatusNotFound, traceID, "not_found", "unknown resource type: "+path[0])
		return
	}

	if err != nil {
		writeError(w, http.StatusNotFound, traceID, "not_found", err.Error())
		return
	}

	data, _ := json.Marshal(result)
	writeJSON(w, http.StatusOK, &provider.ProviderResponse{
		Status:            "ok",
		Data:              data,
		ExternalRequestID: uuid.NewString(),
		TraceID:           traceID,
	})
}

func (h *Handler) executeCapability(ctx context.Context, entityID, capabilityID string, req provider.ProviderRequest) (any, error) {
	switch capabilityID {

	// ── Book & Account Management ──

	case "finance.book.create":
		var input struct {
			Name               string `json:"name"`
			Currency           string `json:"currency"`
			AccountingStandard string `json:"accounting_standard"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		book := &domain.AccountingBook{
			EntityID:           entityID,
			Name:               input.Name,
			BaseCurrency:       input.Currency,
			AccountingStandard: input.AccountingStandard,
		}
		if book.BaseCurrency == "" {
			book.BaseCurrency = "CNY"
		}
		if book.AccountingStandard == "" {
			book.AccountingStandard = "small_business_gaap_cn"
		}
		if err := h.svc.CreateBook(ctx, book); err != nil {
			return nil, err
		}
		return map[string]any{"id": book.ID, "name": book.Name}, nil

	case "finance.book.list":
		books, err := h.svc.ListBooks(ctx, entityID)
		if err != nil {
			return nil, err
		}
		return map[string]any{"books": books}, nil

	case "finance.account.list":
		var input struct {
			Category string `json:"category"`
			Limit    int    `json:"limit"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		if input.Limit <= 0 {
			input.Limit = 200
		}
		result, err := h.svc.ListAccounts(ctx, domain.AccountListQuery{
			EntityID: entityID,
			Category: input.Category,
			Limit:    input.Limit,
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"items": result.Items, "total": result.Total}, nil

	// ── Invoice Workflow ──

	case "finance.invoice.create_draft":
		var inv domain.Invoice
		if err := json.Unmarshal(req.Input, &inv); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, inv.BookID)
		created, err := h.svc.CreateInvoiceDraft(ctx, entityID, bookID, req.Actor.ID, &inv)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"invoice_id": created.ID,
			"status":     created.Status,
		}, nil

	case "finance.invoice.approve":
		var input struct {
			InvoiceID string `json:"invoice_id"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		inv, entry, err := h.svc.ApproveInvoice(ctx, entityID, input.InvoiceID, req.Actor.ID)
		if err != nil {
			return nil, err
		}
		result := map[string]any{
			"invoice_id": inv.ID,
			"status":     inv.Status,
		}
		if entry != nil {
			result["journal_entry_id"] = entry.ID
		}
		return result, nil

	case "finance.invoice.reject":
		var input struct {
			InvoiceID string `json:"invoice_id"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		inv, err := h.svc.RejectInvoice(ctx, entityID, input.InvoiceID, req.Actor.ID)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"invoice_id": inv.ID,
			"status":     inv.Status,
		}, nil

	case "finance.invoice.create_red_letter":
		var input struct {
			BookID            string `json:"book_id"`
			OriginalInvoiceID string `json:"original_invoice_id"`
			RedType           string `json:"red_type"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		inv, err := h.svc.CreateRedLetterInvoice(ctx, entityID, bookID, input.OriginalInvoiceID, req.Actor.ID, input.RedType)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"invoice_id": inv.ID,
			"status":     inv.Status,
		}, nil

	case "finance.invoice.import_einvoice":
		var input struct {
			BookID   string                  `json:"book_id"`
			EInvoice domain.ExtractionResult `json:"einvoice"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		inv, err := h.svc.ImportEInvoice(ctx, entityID, bookID, &input.EInvoice)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"invoice_id": inv.ID,
			"status":     inv.Status,
		}, nil

	case "finance.invoice.verify":
		var input struct {
			InvoiceID string `json:"invoice_id"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		if err := h.svc.VerifyInvoice(ctx, entityID, input.InvoiceID, req.Actor.ID); err != nil {
			return nil, err
		}
		return map[string]any{"status": "verified"}, nil

	case "finance.invoice.confirm_usage":
		var input struct {
			InvoiceID   string `json:"invoice_id"`
			UsageStatus string `json:"usage_status"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		if err := h.svc.ConfirmInvoiceUsage(ctx, entityID, input.InvoiceID, input.UsageStatus, req.Actor.ID); err != nil {
			return nil, err
		}
		return map[string]any{"status": "ok"}, nil

	// ── Journal Workflow ──

	case "finance.journal.create_draft":
		var entry domain.JournalEntry
		if err := json.Unmarshal(req.Input, &entry); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, entry.BookID)
		created, err := h.svc.CreateJournalDraft(ctx, entityID, bookID, entry.Period, req.Actor.ID, &entry)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"journal_entry_id": created.ID,
			"voucher_no":       created.VoucherNo,
			"status":           created.Status,
		}, nil

	case "finance.journal.post":
		var input struct {
			JournalEntryID string `json:"journal_entry_id"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		posted, err := h.svc.PostJournalEntry(ctx, entityID, input.JournalEntryID, req.Actor.ID)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"journal_entry_id": posted.ID,
			"voucher_no":       posted.VoucherNo,
			"status":           posted.Status,
		}, nil

	case "finance.journal.void":
		var input struct {
			JournalEntryID string `json:"journal_entry_id"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		voided, err := h.svc.VoidJournalEntry(ctx, entityID, input.JournalEntryID, req.Actor.ID)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"journal_entry_id": voided.ID,
			"status":           voided.Status,
		}, nil

	case "finance.journal.update_draft":
		var entry domain.JournalEntry
		if err := json.Unmarshal(req.Input, &entry); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		updated, err := h.svc.UpdateJournalDraft(ctx, entityID, entry.ID, &entry)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"journal_entry_id": updated.ID,
			"status":           updated.Status,
		}, nil

	case "finance.journal.batch_post":
		var input struct {
			JournalEntryIDs []string `json:"journal_entry_ids"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		posted, err := h.svc.BatchPostJournals(ctx, entityID, input.JournalEntryIDs, req.Actor.ID)
		if err != nil {
			return nil, err
		}
		return map[string]any{"count": len(posted)}, nil

	// ── Reports ──

	case "finance.report.trial_balance":
		var input struct {
			BookID string `json:"book_id"`
			Period string `json:"period"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		return h.svc.TrialBalance(ctx, entityID, bookID, input.Period)

	case "finance.report.account_balance":
		var input struct {
			BookID string `json:"book_id"`
			Period string `json:"period"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		return h.svc.AccountBalance(ctx, entityID, bookID, input.Period)

	case "finance.report.profit_statement":
		var input struct {
			BookID             string `json:"book_id"`
			Period             string `json:"period"`
			AccountingStandard string `json:"accounting_standard"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		return h.svc.ProfitStatement(ctx, entityID, bookID, input.Period, input.AccountingStandard)

	case "finance.report.balance_sheet":
		var input struct {
			BookID             string `json:"book_id"`
			Period             string `json:"period"`
			AccountingStandard string `json:"accounting_standard"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		return h.svc.BalanceSheet(ctx, entityID, bookID, input.Period, input.AccountingStandard)

	case "finance.report.vat_cross_check":
		var input struct {
			BookID string `json:"book_id"`
			Period string `json:"period"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		return h.svc.VATCrossCheck(ctx, entityID, bookID, input.Period)

	case "finance.report.vat_return":
		var input struct {
			BookID string `json:"book_id"`
			Period string `json:"period"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		return h.svc.VATReturn(ctx, entityID, bookID, input.Period)

	case "finance.report.cross_tax_validation":
		var input struct {
			BookID string `json:"book_id"`
			Period string `json:"period"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		return h.svc.CrossTaxValidation(ctx, entityID, bookID, input.Period)

	case "finance.report.three_way_match":
		var input struct {
			BookID string `json:"book_id"`
			Period string `json:"period"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		return h.svc.ThreeWayMatch(ctx, entityID, bookID, input.Period)

	case "finance.report.tax_risk":
		var input struct {
			BookID string `json:"book_id"`
			Period string `json:"period"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		return h.svc.TaxRisk(ctx, entityID, bookID, input.Period)

	case "finance.report.cit_return":
		var input struct {
			BookID  string `json:"book_id"`
			TaxYear string `json:"tax_year"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		return h.svc.GenerateCITReport(ctx, entityID, bookID, input.TaxYear, req.Actor.ID)

	// ── Period Management ──

	case "finance.period.close_check":
		var input struct {
			BookID string `json:"book_id"`
			Period string `json:"period"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		return h.svc.CloseCheck(ctx, entityID, bookID, input.Period)

	case "finance.period.close":
		var input struct {
			BookID string `json:"book_id"`
			Period string `json:"period"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		p, err := h.svc.ClosePeriod(ctx, entityID, bookID, input.Period, req.Actor.ID)
		if err != nil {
			return nil, err
		}
		return map[string]any{"period": p.Period, "status": p.Status}, nil

	case "finance.period.lock":
		var input struct {
			BookID string `json:"book_id"`
			Period string `json:"period"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		p, err := h.svc.LockPeriod(ctx, entityID, bookID, input.Period, req.Actor.ID)
		if err != nil {
			return nil, err
		}
		return map[string]any{"period": p.Period, "status": p.Status}, nil

	case "finance.period.reopen":
		var input struct {
			BookID string `json:"book_id"`
			Period string `json:"period"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		p, err := h.svc.ReopenPeriod(ctx, entityID, bookID, input.Period, req.Actor.ID)
		if err != nil {
			return nil, err
		}
		return map[string]any{"period": p.Period, "status": p.Status}, nil

	case "finance.period.enhanced_close_check":
		var input struct {
			BookID string `json:"book_id"`
			Period string `json:"period"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		return h.svc.EnhanceCloseCheck(ctx, entityID, bookID, input.Period)

	// ── Reconciliation ──

	case "finance.reconciliation.upsert_logistics":
		var input struct {
			InvoiceID string `json:"invoice_id"`
			WaybillNo string `json:"waybill_no"`
			Carrier   string `json:"carrier"`
			Status    string `json:"status"`
			ShipDate  string `json:"ship_date"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		lr := &domain.LogisticsRecord{
			EntityID:  entityID,
			InvoiceID: input.InvoiceID,
			WaybillNo: input.WaybillNo,
			Carrier:   input.Carrier,
			Status:    input.Status,
			ShipDate:  input.ShipDate,
		}
		if lr.Status == "" {
			lr.Status = domain.LogisticsStatusShipped
		}
		if err := h.svc.UpsertLogistics(ctx, lr); err != nil {
			return nil, err
		}
		return map[string]any{"logistics_id": lr.ID}, nil

	case "finance.reconciliation.upsert_bank_transaction":
		var input struct {
			TransactionDate     string  `json:"transaction_date"`
			CounterpartyName    string  `json:"counterparty_name"`
			CounterpartyAccount string  `json:"counterparty_account"`
			Amount              float64 `json:"amount"`
			Direction           string  `json:"direction"`
			Summary             string  `json:"summary"`
			BankReference       string  `json:"bank_reference"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bt := &domain.BankTransaction{
			EntityID:            entityID,
			TransactionDate:     input.TransactionDate,
			CounterpartyName:    input.CounterpartyName,
			CounterpartyAccount: input.CounterpartyAccount,
			Amount:              input.Amount,
			Direction:           input.Direction,
			Summary:             input.Summary,
			BankReference:       input.BankReference,
		}
		if err := h.svc.UpsertBankTransaction(ctx, bt); err != nil {
			return nil, err
		}
		return map[string]any{"bank_transaction_id": bt.ID}, nil

	case "finance.reconciliation.match_bank":
		var input struct {
			BankTransactionID string  `json:"bank_transaction_id"`
			InvoiceID         string  `json:"invoice_id"`
			Confidence        float64 `json:"confidence"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		if input.Confidence == 0 {
			input.Confidence = 1.0
		}
		if err := h.svc.MatchBankToInvoice(ctx, entityID, input.BankTransactionID, input.InvoiceID, input.Confidence); err != nil {
			return nil, err
		}
		return map[string]any{"status": "matched"}, nil

	case "finance.reconciliation.unmatch_bank":
		var input struct {
			BankTransactionID string `json:"bank_transaction_id"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		if err := h.svc.UnmatchBankFromInvoice(ctx, entityID, input.BankTransactionID); err != nil {
			return nil, err
		}
		return map[string]any{"status": "unmatched"}, nil

	// ── Tax Engine ──

	case "finance.tax.calculate_vat":
		var input struct {
			TaxpayerType string  `json:"taxpayer_type"`
			OutputTax    float64 `json:"output_tax"`
			InputTax     float64 `json:"input_tax"`
			SalesAmount  float64 `json:"sales_amount"`
			LevyRate     float64 `json:"levy_rate"`
			Location     string  `json:"location"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		vatInput := engine.VATInput{
			TaxpayerType: engine.TaxpayerType(input.TaxpayerType),
			OutputTax:    input.OutputTax,
			InputTax:     input.InputTax,
			SalesAmount:  input.SalesAmount,
			LevyRate:     input.LevyRate,
		}
		return h.svc.CalculateVAT(ctx, vatInput, input.Location)

	case "finance.tax.calculate_stamp":
		var input engine.StampTaxInput
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		return h.svc.CalculateStampTax(ctx, input)

	case "finance.tax.calculate_pit":
		var input engine.PITInput
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		return h.svc.CalculatePIT(ctx, input)

	case "finance.tax.list_adjustments":
		var input struct {
			TaxYear  string `json:"tax_year"`
			Category string `json:"category"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		items, _, err := h.svc.ListAdjustments(ctx, domain.AdjustmentListQuery{
			EntityID: entityID,
			TaxYear:  input.TaxYear,
			Category: input.Category,
			Limit:    100,
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"items": items}, nil

	case "finance.tax.upsert_adjustments":
		var input struct {
			BookID      string                    `json:"book_id"`
			TaxYear     string                    `json:"tax_year"`
			Adjustments []domain.AdjustmentRecord `json:"adjustments"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		for i := range input.Adjustments {
			input.Adjustments[i].EntityID = entityID
			input.Adjustments[i].BookID = bookID
			input.Adjustments[i].TaxYear = input.TaxYear
		}
		if err := h.svc.UpsertAdjustments(ctx, input.Adjustments); err != nil {
			return nil, err
		}
		return map[string]any{"status": "ok"}, nil

	// ── Risk & Consistency ──

	case "finance.risk.scan":
		var input struct {
			BookID string `json:"book_id"`
			Period string `json:"period"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		findings, err := h.svc.RiskScan(ctx, entityID, bookID, input.Period)
		if err != nil {
			return nil, err
		}
		return map[string]any{"findings": findings}, nil

	case "finance.consistency.check":
		var input struct {
			BookID string `json:"book_id"`
			Period string `json:"period"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		checks, err := h.svc.RunConsistencyCheck(ctx, entityID, bookID, input.Period)
		if err != nil {
			return nil, err
		}
		return map[string]any{"checks": checks}, nil

	// ── Export ──

	case "finance.export.trial_balance":
		var input struct {
			BookID string `json:"book_id"`
			Period string `json:"period"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		csv, err := h.svc.ExportTrialBalanceCSV(ctx, entityID, bookID, input.Period)
		if err != nil {
			return nil, err
		}
		return map[string]any{"csv": string(csv)}, nil

	case "finance.export.vat_summary":
		var input struct {
			BookID string `json:"book_id"`
			Period string `json:"period"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		csv, err := h.svc.ExportVATSummaryCSV(ctx, entityID, bookID, input.Period)
		if err != nil {
			return nil, err
		}
		return map[string]any{"csv": string(csv)}, nil

	case "finance.export.vat_return":
		var input struct {
			BookID string `json:"book_id"`
			Period string `json:"period"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		data, err := h.svc.ExportVATReturnJSON(ctx, entityID, bookID, input.Period)
		if err != nil {
			return nil, err
		}
		var result any
		json.Unmarshal(data, &result)
		return map[string]any{"data": result}, nil

	case "finance.export.cit_return":
		var input struct {
			BookID  string `json:"book_id"`
			TaxYear string `json:"tax_year"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		bookID, _ := h.svc.ResolveBook(ctx, entityID, input.BookID)
		data, err := h.svc.ExportCITReturnJSON(ctx, entityID, bookID, input.TaxYear, req.Actor.ID)
		if err != nil {
			return nil, err
		}
		var result any
		json.Unmarshal(data, &result)
		return map[string]any{"data": result}, nil

	case "finance.export.adjustments":
		var input struct {
			TaxYear string `json:"tax_year"`
		}
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		data, err := h.svc.ExportAdjustmentsJSON(ctx, entityID, input.TaxYear)
		if err != nil {
			return nil, err
		}
		var result any
		json.Unmarshal(data, &result)
		return map[string]any{"data": result}, nil

	default:
		return nil, fmt.Errorf("unknown capability: %s", capabilityID)
	}
}

func (h *Handler) findCapability(capabilityID string) *capability.Capability {
	for _, c := range h.capabilities {
		if c.Name == capabilityID {
			return &c
		}
	}
	return nil
}

func (h *Handler) getPendingInvoiceCount(ctx context.Context, entityID, bookID string) int {
	if bookID == "" {
		return 0
	}
	invoices, err := h.svc.ListInvoices(ctx, entityID, bookID, "", domain.StatusPendingReview, 1000, 0)
	if err != nil {
		return 0
	}
	return len(invoices)
}

func (h *Handler) getDraftJournalCount(ctx context.Context, entityID, bookID string) int {
	if bookID == "" {
		return 0
	}
	entries, err := h.svc.ListJournalEntries(ctx, domain.JournalListQuery{
		EntityID: entityID,
		Status:   domain.JournalStatusDraft,
		Limit:    1000,
	})
	if err != nil {
		return 0
	}
	return len(entries)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, httpStatus int, traceID, code, message string) {
	writeJSON(w, httpStatus, &provider.ProviderResponse{
		Status: "failed",
		Error: &provider.ProviderError{
			Code:    code,
			Message: message,
		},
		ExternalRequestID: uuid.NewString(),
		TraceID:           traceID,
	})
}
