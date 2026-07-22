> Part of the `testing-guide-hf-income-service` skill (see `../SKILL.md`).

# HTTP Clients (`pkg/httpclient/*.go`)

## What to test

- Success path: correct request built, correct response JSON deserialized into the typed output.
- Timeout: a slow/unresponsive upstream maps to `ErrUpstreamTimeout`, not a generic error.
- Unexpected status code (5xx, unexpected 4xx): maps to `ErrUpstreamError`.
- Any response-shape edge case the real upstream is known to produce (e.g. `data: null` on an empty result — see the `HF-41` fix in `CHANGELOG.md` for a real incident this caused).
- Helper functions with branching, e.g. `LastDayOfMonth` — February, leap years, December rollover.

## Layer assignment

Integration — this is this project's only "external HTTP API" system boundary. Per the fundamentals' Real-vs-Fake table, an external HTTP API is faked (never hit the real `hf-transaction-service` in tests): `httptest.NewServer` gives a real HTTP round-trip without network flakiness or a running dependency.

## Setup pattern

```go
package httpclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/victorgmrqs/hf-income-service/src/pkg/httpclient"
)

func jsonHandler(status int, body any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}
}

func TestGetExpenseTotals_Timeout_ReturnsUpstreamTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond) // longer than the injected client timeout
	}))
	defer srv.Close()

	shortClient := &http.Client{Timeout: 10 * time.Millisecond} // inject a short timeout to trigger it deterministically
	client := httpclient.NewTransactionClient(srv.URL, shortClient)

	_, err := client.GetExpenseTotals(context.Background(), "user-1", "2026-06")
	if !errors.Is(err, httpclient.ErrUpstreamTimeout) {
		t.Fatalf("err = %v, want ErrUpstreamTimeout", err)
	}
}
```

- Always inject the `*http.Client` (constructor takes it) — never rely on the package's real 5s default timeout in a timeout test; inject a short one so the test stays fast and deterministic.
- One `jsonHandler` helper per test file for the success-path fixture; use `http.HandlerFunc` directly for the timeout/error paths since the response never actually gets written to completion.
- `defer srv.Close()` — a leaked `httptest.Server` holds a real OS port until the test binary exits.

## When to skip

- Don't test the underlying `net/http` transport itself (redirects, keep-alive) — that's Go stdlib's job, not this package's contract.

## Examples from project

- `transaction_client_test.go` — `TestGetExpenseTotals_Success`, `_Timeout_ReturnsUpstreamTimeout`, `_UnexpectedStatus_ReturnsUpstreamError`. Same pattern applies to `GetAccountsPayable` and any future endpoint (`GetExpensesByCategory` for MET comparison, per `INTEGRATIONS.md`'s Endpoint 3 placeholder).
