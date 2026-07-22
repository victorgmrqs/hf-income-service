> Part of the `testing-guide-hf-income-service` skill (see `../SKILL.md`).

# Middleware (`pkg/observability/middleware.go`)

## What to test

- `RequestMiddleware` sets the expected span attributes, structured log fields (`trace_id`, `method`, `route`, `status_code`, `duration_ms`), and increments `HTTPRequestDuration`/`HTTPRequestsTotal` with the right labels — for both a successful and a failing downstream handler.
- Trace propagation: an inbound request with a `traceparent` header continues the trace (via `otel.GetTextMapPropagator().Extract`) rather than starting a new one.
- `route == ""` fallback to `"unknown"` for unmatched routes (404s) — otherwise Prometheus label cardinality explodes with raw paths.
- Never asserting the body was logged — `RequestMiddleware` must never log `c.Request.Body`/response bodies (RNF-SEG-02).

## Layer assignment

E2E, through the real router — per the fundamentals' Layer Assignment Table, middleware is tested via accepted/rejected requests, not in isolation. This project's middleware has no auth/rejection logic yet (that will live in a future cross-service `authmiddleware`, see `artifacts/future-types.md`), so a single E2E test asserting the log/metric/span side effects around a real request is sufficient — no additional integration layer needed today.

## Setup pattern

Reuse `router_test.go`'s pattern: build a real router with `RequestMiddleware` attached (it already is, via `SetupRouter`), issue a request, and assert observable side effects — metrics via the real `*ServiceMetrics`' underlying Prometheus registry, not a mock:

```go
metrics := observability.NewServiceMetrics("hf_income_middleware_test")
router := handler.SetupRouter(observability.NewLogger("test"), metrics, incomeHandler, budgetHandler, balanceHandler)

w := httptest.NewRecorder()
router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))

// assert via metrics.HTTPRequestsTotal (prometheus/client_golang/testutil helpers)
// or via a capturing slog.Handler if asserting log fields specifically.
```

- To assert on `slog` output, inject a `slog.NewJSONHandler` writing to a `bytes.Buffer` instead of the default handler, then parse the JSON lines — don't parse `stdout`.
- To assert on Prometheus counters, use `github.com/prometheus/client_golang/prometheus/testutil` (`testutil.ToFloat64(metrics.HTTPRequestsTotal.WithLabelValues(...))`).

## When to skip

- Don't unit-test the middleware function directly by calling it with a hand-built `*gin.Context` — constructing a valid `gin.Context` outside the router's request cycle is fragile and doesn't prove it's wired into the router correctly, which is the actual risk.

## Examples from project

- No dedicated `middleware_test.go` exists yet — `RequestMiddleware` is currently only exercised indirectly through `router_test.go`'s `/health` request, which does not assert on its specific side effects (trace attributes, log fields, metric labels). Add a dedicated E2E test using the pattern above when next touching this file.
