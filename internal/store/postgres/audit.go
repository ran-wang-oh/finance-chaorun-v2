package postgres

import (
	"context"
	"fmt"

	"finance.chao.run/v2/internal/domain"
)

type AuditStore struct {
	db *DB
}

func NewAuditStore(db *DB) *AuditStore {
	return &AuditStore{db: db}
}

func (s *AuditStore) Append(ctx context.Context, e *domain.AuditEntry) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO finance_audit_log (id, entity_id, book_id, capability_id, v2_capability_id,
			actor_type, actor_id, trace_id, workflow_run_id, approval_grant_id,
			idempotency_key, object_type, object_id, action, outcome, payload)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
	`, e.ID, e.EntityID, nullStr(e.BookID), e.CapabilityID, nullStr(e.V2CapabilityID),
		e.ActorType, e.ActorID, e.TraceID, nullStr(e.WorkflowRunID), nullStr(e.ApprovalGrantID),
		nullStr(e.IdempotencyKey), nullStr(e.ObjectType), nullStr(e.ObjectID),
		e.Action, e.Outcome, e.Payload)
	return err
}

func (s *AuditStore) List(ctx context.Context, query domain.AuditListQuery) ([]domain.AuditEntry, error) {
	baseQuery := ` FROM finance_audit_log WHERE entity_id = $1`
	args := []any{query.EntityID}
	argIdx := 2

	if query.BookID != "" {
		baseQuery += ` AND book_id = $` + itoa(argIdx)
		args = append(args, query.BookID)
		argIdx++
	}
	if query.Action != "" {
		baseQuery += ` AND action = $` + itoa(argIdx)
		args = append(args, query.Action)
		argIdx++
	}

	if query.Limit <= 0 {
		query.Limit = 50
	}
	if query.Limit > 200 {
		query.Limit = 200
	}

	selectQuery := `SELECT id, entity_id, COALESCE(book_id, ''), capability_id, COALESCE(v2_capability_id, ''),
		actor_type, actor_id, trace_id, COALESCE(workflow_run_id, ''), COALESCE(approval_grant_id, ''),
		COALESCE(idempotency_key, ''), COALESCE(object_type, ''), COALESCE(object_id, ''),
		action, outcome, payload,
		to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')` + baseQuery

	selectQuery += ` ORDER BY created_at DESC LIMIT $` + itoa(argIdx) + ` OFFSET $` + itoa(argIdx+1)
	args = append(args, query.Limit, query.Offset)

	rows, err := s.db.Pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []domain.AuditEntry
	for rows.Next() {
		var e domain.AuditEntry
		if err := rows.Scan(&e.ID, &e.EntityID, &e.BookID, &e.CapabilityID, &e.V2CapabilityID,
			&e.ActorType, &e.ActorID, &e.TraceID, &e.WorkflowRunID, &e.ApprovalGrantID,
			&e.IdempotencyKey, &e.ObjectType, &e.ObjectID,
			&e.Action, &e.Outcome, &e.Payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
