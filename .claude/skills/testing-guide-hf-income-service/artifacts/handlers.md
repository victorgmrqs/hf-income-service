> Part of the `testing-guide-hf-income-service` skill (see `../SKILL.md`).

# Handlers (`handler/<domain>/*.go`)

## What to test

- Status code + envelope shape for each route: `POST` → `201` + `{"data": {...}, "error": null}`; validation failure → `400` + `{"data": null, "error": {"code": "VALIDATION_ERROR", ...}}`.
- Every branch of `handleError`'s switch — one request per sentinel error, asserting the exact `(status, code)` pair and that `metrics.BusinessErrorsTotal` was incremented for rule violations (see `references/mock-health-rules.md` for how to assert on a real `*ServiceMetrics` without mocking it).
- Param/query parsing failures (`invalid user_id`, `invalid id` on the path) → `400` before the usecase is ever called.
- Route registration order for literal-vs-`:id` conflicts (`/preview-next` before `/:id`) — a regression here silently routes `preview-next` as an ID; the E2E test doubles as the regression guard.

## Layer assignment

E2E only — handlers are a thin parse → call usecase → map response layer with no business logic of its own to unit-test in isolation. Per the fundamentals' Layer Assignment Table: "HTTP handler/controller → ❌ unit (wiring test), ✅ E2E (status codes, validation, access control)". The one piece of real logic (`handleError`'s switch) is best proven through the HTTP surface it actually serves, using a **real usecase whose error you control via a stub**, not by calling the private method directly.

## Setup pattern

Build the router with real handlers and a stub usecase per test (implements the usecase's exported interface, returns a fixed output/error):

```go
package income_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/victorgmrqs/hf-income-service/src/internal/handler/income"
	incomeUseCase "github.com/victorgmrqs/hf-income-service/src/internal/usecase/income"
	"github.com/victorgmrqs/hf-income-service/src/pkg/observability"
)

func init() { gin.SetMode(gin.TestMode) }

type stubCreateUseCase struct {
	out *incomeUseCase.IncomeOutput
	err error
}

func (s *stubCreateUseCase) Execute(_ context.Context, _ incomeUseCase.CreateInput) (*incomeUseCase.IncomeOutput, error) {
	return s.out, s.err
}

func TestIncomeHandler_Create_InvalidAmount(t *testing.T) {
	metrics := observability.NewServiceMetrics("hf_income_handler_test")
	h := income.NewIncomeHandler(
		&stubCreateUseCase{err: incomeUseCase.ErrInvalidAmount}, nil, nil, nil, nil, nil, metrics,
	)
	router := gin.New()
	router.POST("/api/v1/income", h.Create)

	body := `{"user_id":"...","description":"x","amount":"0","date":"2026-06-01","competence":"2026-06","type":"SALARY"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/income", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	var resp struct {
		Error struct{ Code string `json:"code"` } `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error.Code != "INVALID_AMOUNT" {
		t.Errorf("code = %q, want INVALID_AMOUNT", resp.Error.Code)
	}
}
```

- Mount only the route(s) under test on a bare `gin.New()` router — no need to go through the full `SetupRouter` unless testing route-conflict ordering (then use `SetupRouter` exactly as `router_test.go` does).
- One stub struct per usecase interface, same hand-written pattern as `mockIncomeRepository` — see `references/mock-health-rules.md`.
- Use `observability.NewServiceMetrics("<unique-test-namespace>")` — a real instance, never mocked (Prometheus panics on duplicate registration across tests in the same namespace, so vary the namespace per test file if needed).

## When to skip

- Don't write a test per DTO field — one `_VALIDATION_ERROR` scenario per endpoint is enough to prove binding is wired (see fundamentals' "validation passthrough").

## Examples from project

- `router_test.go`'s `TestHealth_ReturnsOK` — the only handler-level test today; full `SetupRouter` + real `httptest` request/response round-trip.
- **Gap:** no test exists yet for `IncomeHandler.Create/List/GetByID/Update/Delete/Propagate`, `GlobalBudgetHandler.*`, or `BalanceHandler.Get`. When touching any of these, add the missing E2E coverage using the pattern above — don't treat "no precedent" as "not required."
