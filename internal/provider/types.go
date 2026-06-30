package provider

import "encoding/json"

type Actor struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// V3InvocationContext is the context envelope sent by V3 HTTPExecutor.
type V3InvocationContext struct {
	EntityID          string `json:"entity_id"`
	ActorID           string `json:"actor_id"`
	AgentID           string `json:"agent_id"`
	RunID             string `json:"run_id"`
	StepID            string `json:"step_id"`
	TraceID           string `json:"trace_id"`
	ApprovalRef       string `json:"approval_ref,omitempty"`
	GrantRef          string `json:"grant_ref,omitempty"`
	CapabilityID      string `json:"capability_id"`
	CapabilityVersion string `json:"capability_version,omitempty"`
	TargetAgentID     string `json:"target_agent_id,omitempty"`
}

// V3Request is the V3 connector envelope: {context, input}.
type V3Request struct {
	Context V3InvocationContext `json:"context"`
	Input   json.RawMessage    `json:"input"`
}

// ToProviderRequest converts a V3 request to the internal ProviderRequest format.
func (r *V3Request) ToProviderRequest(capabilityID string) ProviderRequest {
	return ProviderRequest{
		EntityID: r.Context.EntityID,
		Actor: Actor{
			Type: "agent",
			ID:   r.Context.ActorID,
		},
		Input:          r.Input,
		TraceID:        r.Context.TraceID,
		CapabilityID:   capabilityID,
		IdempotencyKey: r.Context.StepID, // V3 uses step_id for idempotency
	}
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
