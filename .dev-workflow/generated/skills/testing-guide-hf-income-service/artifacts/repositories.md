> Part of the `testing-guide-hf-income-service` skill (see `../SKILL.md`).

# Repositories (`repository/<entity>.go`)

## What to test

- CRUD round-trip against a real PostgreSQL: `Create` persists and generates an ID, `FindByID`/`GetByID` returns the record, `Update` persists changes, `Delete` soft-deletes (row still exists, `deleted_at` set, excluded from subsequent queries).
- Query filters exactly as used in production: `ListByUserAndCompetence`'s `WHERE user_id = ? AND competence = ?`, ordering (`Order("date ASC")`).
- Aggregation correctness: `SumByUserAndCompetence`'s `COALESCE(SUM(amount), 0)` — assert it returns `0` (not an error) when there are no matching rows.
- Uniqueness/constraint behavior: `ExistsByOriginAndCompetence`, and — critically — soft-delete-then-recreate for any entity with a unique index (see `references/gotchas.md`).
- `gorm.ErrRecordNotFound` propagation on `Delete`/`FindByID` for a nonexistent or already-soft-deleted ID.

## Layer assignment

Integration only — no repository method in this project has branching logic; each is a direct GORM query. Per the fundamentals' Layer Assignment Table ("service with no branching, accesses DB → ❌ no unit test, ✅ integration with real DB"), do not write a unit test with a mocked `*gorm.DB`.

## Setup pattern

```go
package repository

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:16-alpine", // pin the tag — never :latest
		tcpostgres.WithDatabase("hf_income_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Skipf("docker/testcontainers indisponível: %v", err) // never t.Fatal here
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	dsn, _ := container.ConnectionString(ctx, "sslmode=disable")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm open: %v", err)
	}
	if err := db.AutoMigrate(&entity.Income{}); err != nil { // only the entities this test needs
		t.Fatalf("automigrate: %v", err)
	}
	return db
}
```

- One `newTestDB`/`setupTestDB` helper per test **file**, not shared across packages — each package's container is independent.
- `t.Skip` (not `t.Fatal`) when Docker/testcontainers is unavailable — keeps local `go test ./...` green without Docker; CI always has Docker.
- `AutoMigrate` only the entities the test actually needs — keeps setup fast and the test's dependencies explicit.
- File suffix `_integration_test.go`, separate from any unit tests in the same package.

## When to skip

- Never — every repository method is a system-boundary contract (a wrong `WHERE` clause, a wrong `COALESCE`, a wrong soft-delete scope is a real bug an integration test is the only thing that catches).

## Examples from project

- `repository/income_integration_test.go` — `TestIncomeRepository_Integration`, one large scenario covering Create → FindByID → List in sequence.
- `repository/global_budget_integration_test.go` — covers `ExistsByUserAndCompetence`, `Upsert`, and the ORC-01 partial unique index behavior after soft delete (see `references/gotchas.md`).
- **Not yet present:** `repository/reduction_goal_integration_test.go` — next up when the MET domain repository is implemented; must cover the `(user_id, category_id, competence)` composite uniqueness the same way `global_budget` does.
