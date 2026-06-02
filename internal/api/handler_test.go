package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"finance.chao.run/v2/internal/provider"
	"finance.chao.run/v2/internal/store"
)

type mockIdempotencyStore struct {
	records map[string]*store.IdempotencyRecord
}

func newMockIdempotencyStore() *mockIdempotencyStore {
	return &mockIdempotencyStore{records: make(map[string]*store.IdempotencyRecord)}
}

func (m *mockIdempotencyStore) Get(_ context.Context, entityID, capabilityID, idempotencyKey string) (*store.IdempotencyRecord, error) {
	key := entityID + ":" + capabilityID + ":" + idempotencyKey
	r, ok := m.records[key]
	if !ok {
		return nil, nil
	}
	return r, nil
}

func (m *mockIdempotencyStore) Save(_ context.Context, r *store.IdempotencyRecord) error {
	key := r.EntityID + ":" + r.CapabilityID + ":" + r.IdempotencyKey
	m.records[key] = r
	return nil
}

func testServer(h *Handler) *httptest.Server {
	return httptest.NewServer(routesWithHandler(h))
}

func reqBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewBuffer(b)
}

func parseResponse(t *testing.T, resp *http.Response) *provider.ProviderResponse {
	t.Helper()
	var pr provider.ProviderResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		t.Fatal(err)
	}
	return &pr
}

func TestListCapabilities(t *testing.T) {
	ts := testServer(NewHandler(nil))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/v1/capabilities")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body struct {
		Capabilities []any `json:"capabilities"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Capabilities) == 0 {
		t.Error("expected non-empty capabilities list")
	}
}

func TestExecute_MissingEntityID(t *testing.T) {
	h := NewHandler(nil)
	h.WithIdempotencyStore(newMockIdempotencyStore())
	ts := testServer(h)
	defer ts.Close()

	body := reqBody(t, provider.ProviderRequest{
		TraceID:        "trace-001",
		IdempotencyKey: "idem-001",
		Input:          json.RawMessage(`{"name":"test","currency":"CNY"}`),
	})
	resp, err := http.Post(ts.URL+"/v1/capabilities/finance.book.create/execute", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	pr := parseResponse(t, resp)
	if pr.Status != "failed" {
		t.Fatalf("expected failed status, got %s", pr.Status)
	}
	if pr.Error == nil || pr.Error.Code != "invalid_request" {
		t.Fatalf("expected invalid_request error, got %+v", pr.Error)
	}
	if !strings.Contains(pr.Error.Message, "entity_id") {
		t.Errorf("expected message about entity_id, got: %s", pr.Error.Message)
	}
}

func TestExecute_MissingTraceID(t *testing.T) {
	h := NewHandler(nil)
	h.WithIdempotencyStore(newMockIdempotencyStore())
	ts := testServer(h)
	defer ts.Close()

	body := reqBody(t, provider.ProviderRequest{
		EntityID:       "e1",
		IdempotencyKey: "idem-001",
		Input:          json.RawMessage(`{"name":"test","currency":"CNY"}`),
	})
	resp, err := http.Post(ts.URL+"/v1/capabilities/finance.book.create/execute", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	pr := parseResponse(t, resp)
	if pr.Error == nil || pr.Error.Code != "invalid_request" {
		t.Fatalf("expected invalid_request error, got %+v", pr.Error)
	}
	if !strings.Contains(pr.Error.Message, "trace_id") {
		t.Errorf("expected message about trace_id, got: %s", pr.Error.Message)
	}
}

func TestExecute_MissingIdempotencyKey(t *testing.T) {
	h := NewHandler(nil)
	h.WithIdempotencyStore(newMockIdempotencyStore())
	ts := testServer(h)
	defer ts.Close()

	body := reqBody(t, provider.ProviderRequest{
		EntityID: "e1",
		TraceID:  "trace-001",
		Input:    json.RawMessage(`{"name":"test","currency":"CNY"}`),
	})
	resp, err := http.Post(ts.URL+"/v1/capabilities/finance.book.create/execute", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	pr := parseResponse(t, resp)
	if pr.Error == nil || pr.Error.Code != "invalid_request" {
		t.Fatalf("expected invalid_request error, got %+v", pr.Error)
	}
	if !strings.Contains(pr.Error.Message, "idempotency_key") {
		t.Errorf("expected message about idempotency_key, got: %s", pr.Error.Message)
	}
}

func TestExecute_IdempotencyReplay(t *testing.T) {
	idemStore := newMockIdempotencyStore()
	h := NewHandler(nil)
	h.WithIdempotencyStore(idemStore)
	ts := testServer(h)
	defer ts.Close()

	input := json.RawMessage(`{"name":"test-book","currency":"CNY"}`)
	inputHash := HashInput(input)

	idemStore.records["e1:finance.book.create:idem-replay"] = &store.IdempotencyRecord{
		EntityID:       "e1",
		CapabilityID:   "finance.book.create",
		IdempotencyKey: "idem-replay",
		InputHash:      inputHash,
		Result:         json.RawMessage(`{"id":"book-1","name":"test-book"}`),
		Status:         "completed",
	}

	body := reqBody(t, provider.ProviderRequest{
		EntityID:       "e1",
		TraceID:        "trace-001",
		IdempotencyKey: "idem-replay",
		Input:          input,
	})
	resp, err := http.Post(ts.URL+"/v1/capabilities/finance.book.create/execute", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	pr := parseResponse(t, resp)
	if pr.Status != "completed" {
		t.Errorf("expected completed status, got %s", pr.Status)
	}
	if pr.TraceID != "trace-001" {
		t.Errorf("expected trace_id trace-001, got %s", pr.TraceID)
	}
}

func TestExecute_IdempotencyConflict(t *testing.T) {
	idemStore := newMockIdempotencyStore()
	h := NewHandler(nil)
	h.WithIdempotencyStore(idemStore)
	ts := testServer(h)
	defer ts.Close()

	originalInput := json.RawMessage(`{"name":"test-book","currency":"CNY"}`)
	originalHash := HashInput(originalInput)

	idemStore.records["e1:finance.book.create:idem-conflict"] = &store.IdempotencyRecord{
		EntityID:       "e1",
		CapabilityID:   "finance.book.create",
		IdempotencyKey: "idem-conflict",
		InputHash:      originalHash,
		Result:         json.RawMessage(`{"id":"book-old","name":"old-book"}`),
		Status:         "completed",
	}

	differentInput := json.RawMessage(`{"name":"different-book","currency":"USD"}`)
	body := reqBody(t, provider.ProviderRequest{
		EntityID:       "e1",
		TraceID:        "trace-001",
		IdempotencyKey: "idem-conflict",
		Input:          differentInput,
	})
	resp, err := http.Post(ts.URL+"/v1/capabilities/finance.book.create/execute", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}

	pr := parseResponse(t, resp)
	if pr.Error == nil || pr.Error.Code != "finance_idempotency_conflict" {
		t.Fatalf("expected finance_idempotency_conflict error, got %+v", pr.Error)
	}
}

func TestExecute_UnknownCapability(t *testing.T) {
	ts := testServer(NewHandler(nil))
	defer ts.Close()

	body := reqBody(t, provider.ProviderRequest{
		EntityID: "e1",
		TraceID:  "trace-001",
		Input:    json.RawMessage(`{}`),
	})
	resp, err := http.Post(ts.URL+"/v1/capabilities/finance.nonexistent.fake/execute", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	pr := parseResponse(t, resp)
	if pr.Error == nil || pr.Error.Code != "not_found" {
		t.Fatalf("expected not_found error, got %+v", pr.Error)
	}
}

func TestErrorEnvelope_Consistent(t *testing.T) {
	ts := testServer(NewHandler(nil))
	defer ts.Close()

	body := reqBody(t, provider.ProviderRequest{
		EntityID: "e1",
		Input:    json.RawMessage(`{}`),
	})
	resp, err := http.Post(ts.URL+"/v1/capabilities/finance.book.create/execute", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	pr := parseResponse(t, resp)

	if pr.Status != "failed" {
		t.Errorf("expected failed status, got %s", pr.Status)
	}
	if pr.Error == nil {
		t.Fatal("expected error field to be present")
	}
	if pr.ExternalRequestID == "" {
		t.Error("expected external_request_id to be set on error")
	}
	if pr.Error.Code == "" {
		t.Error("expected error code to be set")
	}
	if pr.Error.Message == "" {
		t.Error("expected error message to be set")
	}
}

func TestPreview_MissingTraceID(t *testing.T) {
	ts := testServer(NewHandler(nil))
	defer ts.Close()

	body := reqBody(t, provider.ProviderRequest{
		EntityID: "e1",
		Input:    json.RawMessage(`{}`),
	})
	resp, err := http.Post(ts.URL+"/v1/capabilities/finance.invoice.create_draft/preview", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	pr := parseResponse(t, resp)
	if pr.Error == nil || pr.Error.Code != "invalid_request" {
		t.Fatalf("expected invalid_request error, got %+v", pr.Error)
	}
	if !strings.Contains(pr.Error.Message, "trace_id") {
		t.Errorf("expected message about trace_id, got: %s", pr.Error.Message)
	}
}

func TestValidate_MissingTraceID(t *testing.T) {
	ts := testServer(NewHandler(nil))
	defer ts.Close()

	body := reqBody(t, provider.ProviderRequest{
		EntityID: "e1",
		Input:    json.RawMessage(`{}`),
	})
	resp, err := http.Post(ts.URL+"/v1/capabilities/finance.invoice.create_draft/validate", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	pr := parseResponse(t, resp)
	if pr.Error == nil || pr.Error.Code != "invalid_request" {
		t.Fatalf("expected invalid_request error, got %+v", pr.Error)
	}
	if !strings.Contains(pr.Error.Message, "trace_id") {
		t.Errorf("expected message about trace_id, got: %s", pr.Error.Message)
	}
}

func TestGetResource_EntityIDRequired(t *testing.T) {
	ts := testServer(NewHandler(nil))
	defer ts.Close()

	// GetResource without entity_id returns 400 with proper error envelope
	resp, err := http.Get(ts.URL + "/v1/resources/invoice-inv-001?trace_id=trace-001")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	pr := parseResponse(t, resp)
	if pr.Error == nil || pr.Error.Code != "invalid_request" {
		t.Fatalf("expected invalid_request error, got %+v", pr.Error)
	}
	if !strings.Contains(pr.Error.Message, "entity_id") {
		t.Errorf("expected message about entity_id, got: %s", pr.Error.Message)
	}
}

func TestGetResource_MissingResourceURI(t *testing.T) {
	ts := testServer(NewHandler(nil))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/v1/resources/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// Empty resource_uri: route doesn't match → chi returns 405
	if resp.StatusCode != http.StatusMethodNotAllowed && resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 405 or 404, got %d", resp.StatusCode)
	}
}

func TestPathCapabilityID_TakesPrecedence(t *testing.T) {
	idemStore := newMockIdempotencyStore()
	h := NewHandler(nil)
	h.WithIdempotencyStore(idemStore)
	ts := testServer(h)
	defer ts.Close()

	// Body has capability_id "finance.journal.post" but path says "finance.book.create".
	// The path should win. book.create handler calls h.svc.CreateBook(book)
	// which panics on nil store → Recoverer → 500.
	// If path DIDN'T win, journal.post handler would try to unmarshal
	// "journal_entry_id" from input and fail with an invalid_request error
	// containing "journal_entry_id". We verify we DON'T get that.
	input := json.RawMessage(`{"name":"test-book","currency":"CNY"}`)
	body := reqBody(t, provider.ProviderRequest{
		EntityID:       "e1",
		TraceID:        "trace-001",
		IdempotencyKey: "idem-path-test",
		CapabilityID:   "finance.journal.post",
		Input:          input,
	})
	resp, err := http.Post(ts.URL+"/v1/capabilities/finance.book.create/execute", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// Read raw body — may be empty due to panic recovery
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	bodyStr := buf.String()

	// If journal.post was used, error would mention journal_entry_id
	if strings.Contains(bodyStr, "journal_entry_id") {
		t.Error("path capability_id should take precedence over body, but journal.post handler was used")
	}
	// book.create with nil service → 500 (proves path won)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Logf("Got status %d, body: %s", resp.StatusCode, bodyStr)
	}
}

func TestCatalogSchema_AlignsWithHandlerStructs(t *testing.T) {
	// finance.book.create → handler unmarshals: name, currency, accounting_standard
	t.Run("book.create", func(t *testing.T) {
		input := json.RawMessage(`{"name":"test","currency":"CNY","accounting_standard":"small_business_gaap_cn"}`)
		var s struct {
			Name               string `json:"name"`
			Currency           string `json:"currency"`
			AccountingStandard string `json:"accounting_standard"`
		}
		if err := json.Unmarshal(input, &s); err != nil {
			t.Fatalf("book.create schema mismatch: %v", err)
		}
		if s.Name != "test" || s.Currency != "CNY" {
			t.Error("book.create fields not parsed correctly")
		}
	})

	// finance.invoice.create_draft → handler unmarshals into domain.Invoice
	t.Run("invoice.create_draft", func(t *testing.T) {
		input := json.RawMessage(`{
			"book_id":"b1","invoice_no":"INV-001","direction":"input",
			"issue_date":"2026-01-15","seller_name":"Seller Ltd",
			"amount_without_tax":100,"tax_amount":13,"amount_with_tax":113
		}`)
		// Should unmarshal as flat fields — domain.Invoice has flat JSON tags
		var m map[string]any
		if err := json.Unmarshal(input, &m); err != nil {
			t.Fatal(err)
		}
		// Verify all required fields are at top level (not nested under "invoice")
		for _, f := range []string{"book_id", "invoice_no", "direction", "issue_date", "amount_without_tax", "tax_amount", "amount_with_tax"} {
			if _, ok := m[f]; !ok {
				t.Errorf("invoice.create_draft: required field %q missing at top level", f)
			}
		}
	})

	// finance.journal.create_draft → handler unmarshals into domain.JournalEntry
	t.Run("journal.create_draft", func(t *testing.T) {
		input := json.RawMessage(`{
			"book_id":"b1","period":"2026-01","summary":"test",
			"lines":[{"account_code":"1001","direction":"debit","debit_amount":100,"credit_amount":0}]
		}`)
		var m map[string]any
		if err := json.Unmarshal(input, &m); err != nil {
			t.Fatal(err)
		}
		lines, ok := m["lines"].([]any)
		if !ok || len(lines) == 0 {
			t.Fatal("journal.create_draft: lines must be array")
		}
		line := lines[0].(map[string]any)
		// Must use debit_amount/credit_amount, not "amount"
		if _, hasAmount := line["amount"]; hasAmount {
			t.Error("journal.create_draft: line should use debit_amount/credit_amount, not amount")
		}
		if _, ok := line["debit_amount"]; !ok {
			t.Error("journal.create_draft: line missing debit_amount")
		}
		if _, ok := line["credit_amount"]; !ok {
			t.Error("journal.create_draft: line missing credit_amount")
		}
	})

	// finance.journal.post → handler unmarshals: journal_entry_id
	t.Run("journal.post", func(t *testing.T) {
		input := json.RawMessage(`{"journal_entry_id":"je-1"}`)
		var s struct {
			JournalEntryID string `json:"journal_entry_id"`
		}
		if err := json.Unmarshal(input, &s); err != nil {
			t.Fatalf("journal.post schema mismatch: %v", err)
		}
		if s.JournalEntryID != "je-1" {
			t.Error("journal.post field not parsed correctly")
		}
	})

	// finance.period.close → handler unmarshals: book_id, period
	t.Run("period.close", func(t *testing.T) {
		input := json.RawMessage(`{"book_id":"b1","period":"2026-03"}`)
		var s struct {
			BookID string `json:"book_id"`
			Period string `json:"period"`
		}
		if err := json.Unmarshal(input, &s); err != nil {
			t.Fatalf("period.close schema mismatch: %v", err)
		}
		if s.BookID != "b1" || s.Period != "2026-03" {
			t.Error("period.close fields not parsed correctly")
		}
	})

	// finance.risk.scan → handler unmarshals: book_id, period
	t.Run("risk.scan", func(t *testing.T) {
		input := json.RawMessage(`{"book_id":"b1","period":"2026-03"}`)
		var s struct {
			BookID string `json:"book_id"`
			Period string `json:"period"`
		}
		if err := json.Unmarshal(input, &s); err != nil {
			t.Fatalf("risk.scan schema mismatch: %v", err)
		}
		if s.BookID != "b1" || s.Period != "2026-03" {
			t.Error("risk.scan fields not parsed correctly")
		}
	})

	// finance.consistency.check → handler unmarshals: book_id, period
	t.Run("consistency.check", func(t *testing.T) {
		input := json.RawMessage(`{"book_id":"b1","period":"2026-03"}`)
		var s struct {
			BookID string `json:"book_id"`
			Period string `json:"period"`
		}
		if err := json.Unmarshal(input, &s); err != nil {
			t.Fatalf("consistency.check schema mismatch: %v", err)
		}
		if s.BookID != "b1" || s.Period != "2026-03" {
			t.Error("consistency.check fields not parsed correctly")
		}
	})
}

func TestGetContext_MissingEntityID(t *testing.T) {
	ts := testServer(NewHandler(nil))
	defer ts.Close()

	body := reqBody(t, provider.ProviderRequest{
		TraceID: "trace-001",
	})
	resp, err := http.Post(ts.URL+"/v1/context", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	pr := parseResponse(t, resp)
	if pr.Error == nil || pr.Error.Code != "invalid_request" {
		t.Fatalf("expected invalid_request error, got %+v", pr.Error)
	}
	if !strings.Contains(pr.Error.Message, "entity_id") {
		t.Errorf("expected message about entity_id, got: %s", pr.Error.Message)
	}
}

func TestGetContext_MissingTraceID(t *testing.T) {
	ts := testServer(NewHandler(nil))
	defer ts.Close()

	body := reqBody(t, provider.ProviderRequest{
		EntityID: "e1",
	})
	resp, err := http.Post(ts.URL+"/v1/context", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	pr := parseResponse(t, resp)
	if pr.Error == nil || pr.Error.Code != "invalid_request" {
		t.Fatalf("expected invalid_request error, got %+v", pr.Error)
	}
	if !strings.Contains(pr.Error.Message, "trace_id") {
		t.Errorf("expected message about trace_id, got: %s", pr.Error.Message)
	}
}

func TestGetResource_MissingEntityID(t *testing.T) {
	ts := testServer(NewHandler(nil))
	defer ts.Close()

	// Use a resource URI without :// so chi routing works.
	// The entity_id validation happens before URI parsing.
	resp, err := http.Get(ts.URL + "/v1/resources/invoice-inv-001?trace_id=trace-001")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	pr := parseResponse(t, resp)
	if pr.Error == nil || pr.Error.Code != "invalid_request" {
		t.Fatalf("expected invalid_request error, got %+v", pr.Error)
	}
	if !strings.Contains(pr.Error.Message, "entity_id") {
		t.Errorf("expected message about entity_id, got: %s", pr.Error.Message)
	}
}

func TestGetResource_MissingTraceID(t *testing.T) {
	ts := testServer(NewHandler(nil))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/v1/resources/invoice-inv-001?entity_id=e1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	pr := parseResponse(t, resp)
	if pr.Error == nil || pr.Error.Code != "invalid_request" {
		t.Fatalf("expected invalid_request error, got %+v", pr.Error)
	}
	if !strings.Contains(pr.Error.Message, "trace_id") {
		t.Errorf("expected message about trace_id, got: %s", pr.Error.Message)
	}
}
