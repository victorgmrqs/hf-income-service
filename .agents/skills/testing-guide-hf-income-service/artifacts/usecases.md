> Part of the `testing-guide-hf-income-service` skill (see `../SKILL.md`).

# Use Cases (`usecase/<domain>/<operation>.go`)

## What to test

- Every branch of the `Execute` method: required-field checks, format checks (competence `YYYY-MM`, date `YYYY-MM-DD`), business-rule checks (amount > 0, ceiling auto-adjust thresholds), each mapped to its own sentinel error and rule ID.
- The happy path: correct `Output` shape, fields mapped from `Input`/persisted entity.
- For usecases with a system boundary (`auto_adjust`, `preview_next`, `balance.get`): the branch that reacts to the boundary failing (`ErrUpstreamTimeout`/`ErrUpstreamError` mapped correctly), NOT the boundary's own HTTP behavior (that's `artifacts/http-clients.md`).
- For `balance.get` specifically: the `errgroup` all-or-nothing contract — if any of the 4 parallel sources (income sum, budget, expense totals, accounts payable) fails, `Execute` must return an error and `nil` output, never partial data (SAL-05 / FDD-004 §7).
- Idempotency logic (`income.propagate`'s `ExistsByOriginAndCompetence` short-circuit) via the mocked repository returning `true`/`false`.
- Immutability guards (`update` rejecting edits on `origin_id != nil` records — REC-03).

## Layer assignment

Unit only, always — no usecase in this project accesses a real system directly; all access goes through a mocked `repository.<Entity>Repository` and/or a mocked `httpclient.TransactionClient`.

| Characteristic | Layer |
|---|---|
| Branching + repo only (income create/get/list/update/delete, global_budget create/get/update) | Unit — mock repo |
| Branching + repo + `TransactionClient` (auto_adjust, preview_next, balance.get) | Unit — mock repo AND mock client; do not add an integration test per usecase (the client's contract is tested once, see `artifacts/http-clients.md`) |

## Setup pattern

```go
package income

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
)

func TestCreateIncome_Success(t *testing.T) {
	repo := &mockIncomeRepository{
		CreateFn: func(_ context.Context, income *entity.Income) error {
			income.ID = uuid.New() // simulate BeforeCreate
			return nil
		},
	}
	uc := NewCreateUseCase(repo, testLogger())

	out, err := uc.Execute(context.Background(), validCreateInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// assert out fields...
}
```

- Mock struct: one `mock_repository.go` per usecase package, hand-written, with one `XxxFn func(...)` field per interface method you actually exercise (see `references/mock-health-rules.md`).
- Build the happy-path input once via a `valid<X>Input()` helper; each scenario test mutates a single field.
- `testLogger()` returns a real `*slog.Logger` (`observability.NewLogger("test")`) — never a mock.

## When to skip

- Don't write a unit test for a usecase method with no branching and no error path (none currently exist in this project — every `Execute` has at least one validation branch).
- Don't duplicate a scenario already covered by a different mutation of the same input — each test must fail for a distinct reason.

## Examples from project

- `income.createUseCase` — 6 scenarios (`TestCreateIncome_Success`, `_MissingRequiredField`, `_InvalidAmount`, `_InvalidIncomeType`, `_InvalidCompetence`, implicit `_InvalidDate`) — pure branching + repo, no system boundary.
- `global_budget.autoAdjustUseCase` — branching (ORC-03/04) + repo + `TransactionClient` mock — covers `ErrNoPreviousBudget`, `ErrUpstreamUnavailable`, and both auto-adjust outcomes.
- `balance.getUseCase` — 4-source `errgroup` fan-out; test scenarios include full success, no budget registered (ORC-06/07 nulls), and each of the 4 sources failing independently.
