# Gemini Code Assistant Context: hf-income-service

## Project Overview

This is the **hf-income-service**, a Go REST API that manages income (salaries, freelance, etc.), monthly spending ceilings, balance calculation, and spending reduction goals for the Home Finance ecosystem.

### Key Technologies
- **Go 1.23+** with **Gin** HTTP framework
- **GORM** with **PostgreSQL** driver
- **Clean Architecture**: `entity → repository → usecase → handler`
- **shopspring/decimal** for monetary values (never float64)
- **google/uuid** for UUID generation

### Core Concepts
- **Income (REC):** Recurring or one-time revenue entries per user per competence (YYYY-MM)
- **GlobalBudget (ORC):** Monthly spending ceiling with auto-adjustment logic
- **Balance (SAL):** Computed view — aggregates income from this service + expenses from hf-transaction-service
- **ReductionGoal (MET):** Monthly per-category spending targets with comparatives

**External dependency:** This service calls `hf-transaction-service` API to fetch expense totals and accounts payable for balance calculation.

## Building and Running

### Prerequisites
- Go 1.23+
- PostgreSQL 16+
- Running `hf-transaction-service` (for balance endpoints)

### Running Locally
1. Copy `.env.example` to `.env` and fill in the values
2. `go mod tidy`
3. `go run ./cmd/server`

Server starts on `APP_PORT` (default `8081`).

### Building with Docker
```sh
docker build -t hf-income-service .
docker run -p 8081:8081 --env-file .env hf-income-service
```

## Development Conventions

### Architecture
Follow the dependency rule strictly:
```
Handler → UseCase interface → Repository interface → GORM
Handler → UseCase interface → HTTPClient interface → hf-transaction-service
```
- One use case per file: `create.go`, `get.go`, `update.go`, `delete.go`
- Each use case defines its own `Input`, `Output`, and `Execute()` method
- Handlers only parse HTTP and call use cases — zero business logic in handlers
- Monetary values always use `decimal.Decimal`, never `float64`

### Business Rules — Source of Truth
**The Notion workspace is the single source of truth.**
- Notion: https://www.notion.so/30ee65ad4ba98086809ed3a3f38ef45f
- Local mirror: `RULES.md` — in case of divergence, Notion prevails

Reference rule IDs in error variables:
```go
// REC-03
var ErrCannotEditPastIncome = errors.New("income edits apply from current competence forward only")
```

### Definition of Done (DoD)
Every business rule change must:
1. Reference the rule ID in the code comment
2. Update `RULES.md` status (✅ / 🚧 / 📅)
3. Update Notion
4. Reference the rule ID in PR under "Regras Afetadas"

### API and Testing
- All responses: `{ "data": ..., "error": null }` or `{ "data": null, "error": { "code": "...", "message": "..." } }`
- Consult `openapi.yaml` for the complete API specification
- Repository integration tests use `testcontainers-go` (real PostgreSQL)
- HTTPClient calls to `hf-transaction-service` are abstracted behind an interface for unit testing

### How to Run Tests
```sh
go test ./...
go test -cover ./...
go test -run TestIncomeUseCase_Create ./internal/usecase/income/
```
