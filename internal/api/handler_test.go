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
