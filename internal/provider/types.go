package provider

import "encoding/json"

type Actor struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type ProviderRequest struct {
	EntityID       string          `json:"entity_id"`
	CapabilityID   string          `json:"capability_id"`
	Actor          Actor           `json:"actor"`
	Input          json.RawMessage `json:"input"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	TraceID        string          `json:"trace_id"`
	DryRun         bool            `json:"dry_run"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

type ProviderResponse struct {
	Status             string          `json:"status"`
	Data               json.RawMessage `json:"data,omitempty"`
	ResourceRefs       []string        `json:"resource_refs,omitempty"`
	ExternalRequestID  string          `json:"external_request_id,omitempty"`
	TraceID            string          `json:"trace_id,omitempty"`
	DomainAuditRef     string          `json:"domain_audit_ref,omitempty"`
	Warnings           []string        `json:"warnings,omitempty"`
	Error              *ProviderError  `json:"error,omitempty"`
}

type ProviderError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type ActorType string

const (
	ActorTypeUser  ActorType = "user"
	ActorTypeAgent ActorType = "agent"
)
