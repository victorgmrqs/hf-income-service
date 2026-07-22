> Part of the testing-guide-hf-income-service skill (see ../SKILL.md).

# Gotchas

## `NewServiceMetrics` panics on the second call with the same namespace, in the same test binary

`observability.NewServiceMetrics(namespace string)` uses `promauto`, which registers every metric on Prometheus's **global default registry** — its own doc comment says "panics if called more than once." `router_test.go` already calls it once (`NewServiceMetrics("hf_income_router_test")`). If you add a second test *in the same package* (e.g. new handler E2E tests per `artifacts/handlers.md`) that calls `NewServiceMetrics("hf_income_router_test")` again, the test binary panics with `duplicate metrics collector registration attempted` — even though each `go test` invocation runs in a fresh process, all tests within one package share that process.

**Fix:** give every `NewServiceMetrics` call in a package a unique namespace string, or build it once per package (a package-level `sync.OnceValue`/`TestMain`-initialized var) and reuse it across that package's test functions — don't call `NewServiceMetrics` fresh inside every test function once more than one exists per package.

## testcontainers wait strategy: use `ForListeningPort`, not `ForLog`

The project's actual integration tests (`repository/income_integration_test.go`) wait with `wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second)` — a previously-generated example in an old skill used `wait.ForLog("database system is ready to accept connections").WithOccurrence(2)` instead, which is more fragile (log-message wording is an implementation detail of the Postgres image). Stick with `ForListeningPort` for new integration tests.

## Always pin the Postgres image tag

Use `postgres:16-alpine` (as the project already does) — never `postgres:latest`. An unpinned tag makes test failures non-reproducible when the image updates upstream.

## Soft delete + unique index: the constraint needs to be partial

A plain `uniqueIndex` GORM tag on a soft-deleted entity's business key breaks the moment a row is deleted and recreated — the "deleted" row still occupies the unique slot. This project hit this for real: `GlobalBudget` originally shipped with no soft delete (`HF-59`), later needed it, and the naive unique index on `(user_id, competence)` had to become a **partial** unique index — `(user_id, competence) WHERE deleted_at IS NULL` — via a dedicated `EnsureGlobalBudgetIndexes` helper (see `ADR-001` and the `HF-44` changelog entry). Any new entity combining soft delete with a uniqueness rule (the upcoming `ReductionGoal`'s `(user_id, category_id, competence)`, per MET-04) needs the same partial-index treatment from day one, plus an integration test that does create → soft-delete → recreate and asserts it succeeds — not just a single happy-path create test.

## `httptest.Server` + injected timeout, not `context.WithTimeout`, for deterministic upstream-timeout tests

To make `ErrUpstreamTimeout` fire reliably and fast in a test, inject a short-timeout `*http.Client` into `NewTransactionClient` (see `artifacts/http-clients.md`) rather than relying on the production 5s timeout — a test that waits out a real 5s timeout is slow and, on a loaded CI runner, can flake in the other direction (occasionally not timing out fast enough relative to the test's own assumptions).

## `t.Skip`, not `t.Fatal`, when Docker is unavailable

Every integration test in this project skips (doesn't fail) when `testcontainers.Run` errors, so `go test ./...` stays green on a machine without Docker. Keep this pattern for any new integration test — don't turn a missing local dependency into a false-negative test failure.

## `decimal.Decimal` equality: use `.Equal()`, never `==`

`shopspring/decimal` values with the same numeric value can have different internal representations (e.g. `"5000"` vs `"5000.00"`); `==` compares the struct internals, not the numeric value. Every assertion in this codebase already uses `got.Amount.Equal(want)` — keep doing that, not `got.Amount == want`.
