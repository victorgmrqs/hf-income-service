# CLAUDE.md

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
- `config/` — Viper-based config loading from `.env`

Routing is set up in `internal/handler/routes.go` under `/api/v1/` with RESTful resource groups.

## Domain Entities

| Entity | Domain | Description |
|---|---|---|
| `Income` | REC | Income entries per user/competence (salary, freelance, etc.) |
| `GlobalBudget` | ORC | Monthly spending ceiling per user |
| `ReductionGoal` | MET | Spending reduction goals per category/competence |

Balance (SAL) is a computed response — no persistence. It aggregates income from this service and expenses from `hf-transaction-service`.

## Business Rules — Source of Truth

**The Notion workspace is the single source of truth for all business rules.**

- **Notion (business rules):** https://www.notion.so/30ee65ad4ba98086809ed3a3f38ef45f
- **Local mirror:** `RULES.md` in this repository reflects Notion content. In case of divergence, Notion prevails.

Every error variable or validation that enforces a business rule must reference its rule ID in a comment:
```go
// REC-03
var ErrCannotEditPastIncome = errors.New("income edits apply from current competence forward only")
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
2. **Update `RULES.md`** in this repository — add or update the rule row with correct status
3. **Update Notion** — add or update the rule in the corresponding domain page
4. **Reference the rule ID in the PR description** under "Regras Afetadas"

If a rule does not yet exist in Notion, document it there **before** implementing it in code.
