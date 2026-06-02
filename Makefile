.PHONY: start stop start-postgres wait-ready verify test test-integration test-all logs clean

start-postgres:
	docker compose up -d finance-postgres
	@echo "Waiting for postgres to be healthy..."
	@until docker compose exec -T finance-postgres pg_isready -U chaorun -d chaorun_finance 2>/dev/null; do sleep 0.5; done
	@echo "Postgres ready on :5433"

start: start-postgres
	docker compose up -d finance-provider
	@echo "Provider starting on :8082..."

stop:
	docker compose down

logs:
	docker compose logs -f

wait-ready:
	@echo "Waiting for readyz..."
	@for i in $$(seq 1 30); do \
		status=$$(curl -s -o /dev/null -w '%{http_code}' http://localhost:8082/readyz 2>/dev/null); \
		if [ "$$status" = "200" ]; then \
			echo "Ready!"; \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "Timed out waiting for readyz"; \
	exit 1

verify: wait-ready
	@echo "=== List Capabilities ==="
	@curl -s http://localhost:8082/v1/capabilities | jq '.capabilities | length'
	@echo ""
	@echo "=== List Books ==="
	@curl -s -X POST http://localhost:8082/v1/context \
		-H 'Content-Type: application/json' \
		-d '{"entity_id":"default","trace_id":"smoke-001"}' | jq .
	@echo ""
	@echo "=== Create Journal Draft ==="
	@curl -s -X POST http://localhost:8082/v1/capabilities/finance.journal.create_draft/execute \
		-H 'Content-Type: application/json' \
		-d '{"entity_id":"default","trace_id":"smoke-002","idempotency_key":"smoke-je-1","input":{"book_id":"book-default","period":"2026-06","summary":"smoke test journal","lines":[{"account_code":"1001","direction":"debit","debit_amount":100,"credit_amount":0},{"account_code":"6001","direction":"credit","debit_amount":0,"credit_amount":100}]}}' | jq .
	@echo ""
	@echo "=== Post Journal ==="
	@JE_ID=$$(curl -s -X POST http://localhost:8082/v1/capabilities/finance.journal.create_draft/execute \
		-H 'Content-Type: application/json' \
		-d '{"entity_id":"default","trace_id":"smoke-003","idempotency_key":"smoke-je-2","input":{"book_id":"book-default","period":"2026-06","summary":"post test","lines":[{"account_code":"1001","direction":"debit","debit_amount":50,"credit_amount":0},{"account_code":"5602","direction":"credit","debit_amount":0,"credit_amount":50}]}}' | jq -r '.data.id // empty'); \
	if [ -n "$$JE_ID" ]; then \
		echo "Created journal: $$JE_ID"; \
		curl -s -X POST http://localhost:8082/v1/capabilities/finance.journal.post/execute \
			-H 'Content-Type: application/json' \
			-d "{\"entity_id\":\"default\",\"trace_id\":\"smoke-004\",\"idempotency_key\":\"smoke-post-1\",\"input\":{\"journal_entry_id\":\"$$JE_ID\"}}" | jq .; \
	fi
	@echo ""
	@echo "=== Trial Balance ==="
	@curl -s -X POST http://localhost:8082/v1/capabilities/finance.report.trial_balance/execute \
		-H 'Content-Type: application/json' \
		-d '{"entity_id":"default","trace_id":"smoke-005","input":{"book_id":"book-default","period":"2026-06"}}' | jq .
	@echo ""
	@echo "=== Smoke Complete ==="

test:
	go test -count=1 ./internal/...

test-integration:
	@echo "Running integration tests (requires DATABASE_DSN)..."
	@DATABASE_DSN="postgres://chaorun:chaorun_dev@localhost:5433/chaorun_finance?sslmode=disable" \
		go test -count=1 -v ./internal/store/postgres/... -run 'Test.*Store|Test.*Balance|Test.*Audit'

test-all: test test-integration

clean:
	docker compose down -v
