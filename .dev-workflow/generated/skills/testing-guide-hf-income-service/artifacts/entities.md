> Part of the `testing-guide-hf-income-service` skill (see `../SKILL.md`).

# Entities (`entity/<name>.go`)

## What to test

- `BeforeCreate` hook: generates a UUID when `ID == uuid.Nil`, preserves an already-set `ID` — pure logic, no DB needed.
- DB-level constraints that only a real database enforces: unique indexes (composite or partial), NOT NULL, column types (`decimal(12,2)` actually rejecting/rounding correctly), soft-delete scoping (`deleted_at IS NULL` applied automatically by GORM's default scope).
- Soft-delete-then-recreate for any entity with a unique index — see `references/gotchas.md` for why this is not automatic.

## Layer assignment

Split — this is one of the few artifact types in this project that needs both:

| What | Layer | Why |
|---|---|---|
| `BeforeCreate` hook logic | Unit | Pure function of the struct's `ID` field; no DB call, `tx *gorm.DB` param is unused (pass `nil`) |
| Unique/composite index behavior, NOT NULL, soft delete scoping | Integration | Only a real Postgres enforces/exposes these — GORM struct tags describe intent, the database enforces it |

## Setup pattern

Unit (hook):

```go
package entity_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
)

func TestIncome_BeforeCreate_GeneratesUUID(t *testing.T) {
	income := &entity.Income{}
	if err := income.BeforeCreate(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if income.ID == uuid.Nil {
		t.Error("expected UUID to be generated, got nil")
	}
}
```

Integration (constraints) — reuse the `newTestDB` pattern from `artifacts/repositories.md`, then assert on the raw error from a constraint violation (e.g. a second `Create` with the same `(user_id, competence)` after the first was soft-deleted).

## When to skip

- Don't unit-test struct field presence/types (`income.UserID` is a `uuid.UUID` — the compiler already guarantees that).
- Don't integration-test an entity with no constraints beyond the primary key — a plain `BeforeCreate` unit test is enough.

## Examples from project

- `entity_test.go` — `TestIncome_BeforeCreate_GeneratesUUID`, `TestIncome_BeforeCreate_PreservesExistingUUID`, `TestGlobalBudget_BeforeCreate_GeneratesUUID` — unit, no DB.
- `entity_integration_test.go` — constraint-level tests against real Postgres (unique indexes, soft delete).
- **Not yet present:** `ReductionGoal` — the struct exists (`entity/reduction_goal.go`, MET-04 composite unique index `(user_id, category_id, competence)`) but has no dedicated test yet; add both a `BeforeCreate` unit test and a composite-uniqueness integration test when the MET domain repository ships.
