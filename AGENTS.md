# AGENTS.md

> Arquivo de instruções cross-tool (Claude Code, Antigravity CLI/agy, Cursor).
> Instruções específicas por ferramenta: `CLAUDE.md` (Claude) | `GEMINI.md` (agy).

## Project Overview

Home Finance Income Service — Go REST API responsible for income tracking, monthly global budget ceiling, monthly balance calculation, and spending reduction goals. Part of the Home Finance ecosystem.

**Sister service:** `hf-transaction-service` — manages expenses, categories, payment methods. This service consumes its API to calculate balances and goal progress.

## Common Commands

```bash
# Build
go build ./cmd/server

# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/usecase/income/

# Run a single test by name
go test -run TestIncomeUseCase_Create ./internal/usecase/income/

# Run tests with coverage
go test -cover ./...

# Run the application (requires PostgreSQL and .env)
go run ./cmd/server
```

## Architecture

Clean architecture with four layers following the dependency rule `Handler → UseCase → Repository (interface) → DB`:

- **entity** (`internal/entity/`) — Domain models with GORM tags. UUID v4 primary keys generated in `BeforeCreate` hooks. Soft deletes via `gorm.DeletedAt`.
- **repository** (`internal/repository/`) — Data access. Interfaces defined in `interfaces.go`, implementations in per-entity files. Uses GORM.
- **usecase** (`internal/usecase/<domain>/`) — Business logic. One operation per file (`create.go`, `get.go`, `update.go`, `delete.go`). Each defines its own interface, input/output types, and `Execute()` method.
- **handler** (`internal/handler/<domain>/`) — Gin HTTP handlers. Receives usecase interfaces via constructor injection.

Supporting packages:
- `pkg/response/` — Standardized JSON responses: `response.Success(c, status, data)` and `response.Error(c, status, code, message)`
- `pkg/database/` — PostgreSQL connection via GORM
- `pkg/httpclient/` — HTTP client for calls to `hf-transaction-service`
- `pkg/observability/` — Logger (slog), Prometheus metrics, OTEL tracer, request middleware
- `config/` — Viper-based config loading from `.env`

Routing is set up in `internal/handler/routes.go` under `/api/v1/` with RESTful resource groups.

## Domain Entities

| Entity | Domain | Description |
|---|---|---|
| `Income` | REC | Income entries per user/competence (salary, freelance, etc.) |
| `GlobalBudget` | ORC | Monthly spending ceiling per user |
| `ReductionGoal` | MET | Spending reduction goals per category/competence |

Balance (SAL) is a computed response — no persistence. It aggregates income from this service and expenses from `hf-transaction-service`.

## Requirements

- **Functional & Non-functional:** [`docs/requirements.md`](docs/requirements.md) — RF per domain (with priority and traceability) + RNF (performance targets, availability, security, observability, maintainability)

## Development Workflow

- **Process & Jira template:** [`docs/workflow.md`](docs/workflow.md) — ticket template, Epic structure, Label conventions, `.http` evidence files, branch/commit naming, AI agent usage guide
- **Jira project:** `HF` — https://goncalvesmarques.atlassian.net
- **Implementation plan:** [`docs/implementation-plan.md`](docs/implementation-plan.md) — T01–T19 ordered tasks with estimates and dependencies

## Business Rules — Source of Truth

**Confluence is the single source of truth for all business rules.**

- **Confluence:** https://goncalvesmarques.atlassian.net/wiki (Home Finance space)
- **Local mirror:** `docs/rules/` — one file per domain. In case of divergence, Confluence prevails.

| Domain | File | Description |
|--------|------|-------------|
| REC | [`docs/rules/REC.md`](docs/rules/REC.md) | Income entries |
| ORC | [`docs/rules/ORC.md`](docs/rules/ORC.md) | Global budget ceiling |
| SAL | [`docs/rules/SAL.md`](docs/rules/SAL.md) | Monthly balance |
| MET | [`docs/rules/MET.md`](docs/rules/MET.md) | Reduction goals |

Every error variable or validation that enforces a business rule must reference its rule ID in a comment:
```go
// REC-03
var ErrCannotEditPropagatedIncome = errors.New("propagated income cannot be edited")
```

## Observability Conventions

Instrumentation lives in `pkg/observability/`. Full reference: [`docs/observability.md`](docs/observability.md).

**Rules enforced in every code change:**

- Every new use case logs `INFO` on entry (operation, user_id, competence) and on exit (duration_ms)
- Business rule violations log `WARN` with the rule ID: `slog.String("rule_id", "REC-02")`
- Every error that reaches the handler is logged `ERROR` with `trace_id`
- `BusinessErrorsTotal` metric is incremented on every domain error: `metrics.BusinessErrorsTotal.WithLabelValues("REC", "REC-02").Inc()`

**Never log:**
- Passwords, tokens, API keys, DB connection strings
- Request/response bodies
- `Authorization` or `OTEL_EXPORTER_OTLP_HEADERS` values

**Always include in log statements:**
```go
slog.String("trace_id", traceID)  // from span context — correlates with Grafana traces
```

## Testing Patterns

- Unit tests for use cases: mock repository interfaces
- Integration tests for repositories: use real PostgreSQL via `testcontainers-go`
- No mocking of `hf-transaction-service` in unit tests — use interface abstraction for the HTTP client

## Key Dependencies

```
github.com/gin-gonic/gin       — HTTP framework
gorm.io/gorm                   — ORM
gorm.io/driver/postgres        — PostgreSQL driver
github.com/google/uuid         — UUID generation
github.com/shopspring/decimal  — Precise decimal arithmetic
github.com/spf13/viper         — Configuration management
```

## Definition of Done (DoD)

Every code change that adds or modifies a business rule MUST:

1. **Reference the rule ID** in the error variable or validation comment:
   ```go
   // ORC-03
   var ErrCeilingAutoAdjusted = errors.New("ceiling auto-adjusted from previous month spending")
   ```
2. **Update `docs/rules/<DOMAIN>.md`** — add or update the rule row with correct status
3. **Update Confluence** — add or update the rule in the corresponding domain page
4. **Reference the rule ID in the PR description** under "Regras Afetadas"

If a rule does not yet exist in Confluence, document it there **before** implementing it in code.
