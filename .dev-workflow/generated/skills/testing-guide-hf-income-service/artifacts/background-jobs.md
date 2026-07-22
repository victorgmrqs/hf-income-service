> Part of the `testing-guide-hf-income-service` skill (see `../SKILL.md`).

# Background Jobs (`internal/scheduler/*.go`)

Go-specific artifact type: a goroutine driven by a `time.Ticker`, dependency-injected with usecase interfaces, not present in typical controller/service-oriented frameworks. This project currently has one instance: the monthly scheduler (`HF-38`) that fires `income.PropagateUseCase` and `global_budget.AutoAdjustUseCase` on the 1st of the month.

## What to test

- Trigger condition: the job fires on the configured day/condition, not on every tick.
- Dedupe: the job does not re-run for a competence it already processed in the same run window.
- Per-unit error isolation: one user's `AutoAdjustUseCase` failing must not stop the loop for the remaining users — assert all users were still attempted and the error was logged, not propagated.
- Timezone handling: an invalid `SCHEDULER_TZ` falls back to UTC with a WARN log rather than crashing.
- Graceful shutdown: the goroutine stops cleanly on context cancellation (`signal.NotifyContext`) without leaking.

## Layer assignment

Unit — the scheduler's own logic (when to fire, dedupe, error isolation) has no direct system boundary of its own; it delegates to usecases that are mocked in this test the same way a usecase test mocks a repository. The usecases' own system boundaries (DB, HTTP) are covered by their own tests, not re-verified here.

## Setup pattern

```go
package scheduler

import (
	"context"
	"sync"
	"testing"

	incomeUseCase "github.com/victorgmrqs/hf-income-service/src/internal/usecase/income"
)

// fakePropagate implements income.PropagateUseCase with thread-safe call recording —
// the scheduler runs its loop in a goroutine, so plain slice/int fields would race.
type fakePropagate struct {
	mu    sync.Mutex
	calls []string
	err   error
}

func (f *fakePropagate) Execute(_ context.Context, input incomeUseCase.PropagateInput) (*incomeUseCase.PropagateOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, input.Competence)
	if f.err != nil {
		return nil, f.err
	}
	return &incomeUseCase.PropagateOutput{Propagated: 2}, nil
}
```

- Every fake shared between the test goroutine and the scheduler's own goroutine needs a `sync.Mutex` — this is the one artifact type in the project where a plain unguarded field is a real (flaky, non-deterministic) bug, not just a style nit.
- Inject the tick interval / clock rather than relying on the real 1h ticker in tests — drive the loop directly or use a short interval with a bounded wait.
- Consider Go 1.25's `testing/synctest` (virtual time bubble) if a test needs precise ordering across multiple simulated ticks without real sleeps — not yet used in this project, but a good fit for this artifact type specifically.

## When to skip

- Don't test the injected usecases' internal logic here — that belongs in `artifacts/usecases.md`. This layer only tests the scheduler's own orchestration (when/how often/error isolation).

## Examples from project

- `scheduler_test.go` — `fakePropagate`/`fakeAutoAdjust` with per-user `errFor` injection to test error isolation; dedupe-by-competence and TZ fallback scenarios.
