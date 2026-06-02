package provider

import "context"

type contextKey string

const (
	TraceIDContextKey         contextKey = "provider_trace_id"
	CapabilityIDContextKey    contextKey = "provider_capability_id"
	IdempotencyKeyContextKey  contextKey = "provider_idempotency_key"
	ActorIDContextKey         contextKey = "provider_actor_id"
	ApprovalGrantIDContextKey contextKey = "provider_approval_grant_id"
)

func ContextWithTraceID(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, TraceIDContextKey, v)
}

func TraceIDFromContext(ctx context.Context) string {
	s, _ := ctx.Value(TraceIDContextKey).(string)
	return s
}

func ContextWithCapabilityID(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, CapabilityIDContextKey, v)
}

func CapabilityIDFromContext(ctx context.Context) string {
	s, _ := ctx.Value(CapabilityIDContextKey).(string)
	return s
}

func ContextWithIdempotencyKey(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, IdempotencyKeyContextKey, v)
}

func IdempotencyKeyFromContext(ctx context.Context) string {
	s, _ := ctx.Value(IdempotencyKeyContextKey).(string)
	return s
}

func ContextWithActorID(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, ActorIDContextKey, v)
}

func ActorIDFromContext(ctx context.Context) string {
	s, _ := ctx.Value(ActorIDContextKey).(string)
	return s
}

func ContextWithApprovalGrantID(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, ApprovalGrantIDContextKey, v)
}

func ApprovalGrantIDFromContext(ctx context.Context) string {
	s, _ := ctx.Value(ApprovalGrantIDContextKey).(string)
	return s
}
