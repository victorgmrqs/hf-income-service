---
name: testing-guide-hf-income-service
description: >
  Testing guide for hf-income-service. Reference this skill when planning features,
  implementing code, creating tests, or reviewing changes in hf-income-service.
  Covers what to test, at which layer, and how to set up each test —
  organized by artifact type.
  Triggers on: planning hf-income-service features, implementing hf-income-service
  features, writing tests for hf-income-service, reviewing hf-income-service code,
  reviewing hf-income-service tests, what should I test in hf-income-service,
  how to test hf-income-service, hf-income-service test guide.
---

# Testing Guide — hf-income-service

## 0. Purpose

This guide helps you decide **what to test**, at **which layer**, and **how to set up tests** for each type of artifact in `hf-income-service`. When working on a specific artifact type (usecase, repository, entity, handler, HTTP client, background job, middleware...), read the corresponding guide in `artifacts/` for the complete recipe. Supporting references (mock strategies, file conventions, gotchas) are in `references/`.

## 1. Testability Foundations

- **Clean architecture makes the mock boundary obvious.** `Handler → UseCase → Repository (interface) → DB`. Usecases depend on `repository.<Entity>Repository` interfaces and `httpclient.TransactionClient` — both are the natural mock boundary for unit tests, exactly because each has its own dedicated test suite (repository → integration tests with real Postgres; `TransactionClient` → integration tests with a fake HTTP server). A unit test that mocks `IncomeRepository` proves `createUseCase`'s branching is correct; it says nothing about whether the SQL is correct — that's what `repository/income_integration_test.go` is for. Neither substitutes the other.
- **Repositories have no unit tests, by design.** Every repository method in this project is a thin GORM query with no branching (see `artifacts/repositories.md`) — per the fundamentals' Layer Assignment Table, "service with no branching, accesses DB" skips unit entirely and goes straight to integration with a real database. Mocking `*gorm.DB` would only prove the mock was called, not that the query is correct.
- **`TransactionClient` is the system boundary for hf-transaction-service, not each usecase that calls it.** Usecases (`auto_adjust`, `preview_next`, `balance.get`) unit-test their own branching with a mocked `TransactionClient`; the HTTP contract itself (timeouts, status mapping, JSON shape) is proven once, in `pkg/httpclient/transaction_client_test.go`, against a fake `httptest.NewServer`. Don't re-test the HTTP contract inside every usecase that happens to call the client.
- **Configured libs get real instances, not mocks.** `slog.Logger` and `observability.ServiceMetrics` are constructed for real in tests (`observability.NewLogger("test")`, `observability.NewServiceMetrics("...")`) rather than mocked — see `router_test.go` and `scheduler_test.go`. A mocked logger would never catch a wrong `slog.String` key or a metric label mismatch.
- **Module compilation has no direct equivalent here.** Go's compiler already catches wiring errors that a DI-framework project only catches at runtime (missing import, wrong type) — `go build ./src/cmd/server` (the `build` command in `workflow.config.yaml`) is the compile-time safety net; there is no separate "does it wire up" test layer to write.
- **Soft delete + unique constraints is a known GORM sharp edge this project already hit.** A naive unique index on `(user_id, competence)` breaks the moment a row is soft-deleted and recreated — see `ADR-001` and `references/gotchas.md`. Any new entity that combines soft delete with a uniqueness rule needs an integration test proving delete-then-recreate works, not just a happy-path create test.
- **Go 1.25 has `testing/synctest`** for deterministic concurrent tests using virtual time instead of real `time.Sleep`. The scheduler's tests currently use real goroutines + `sync.Mutex` + short real sleeps; `synctest` is a candidate for removing that real-time dependency if scheduler tests become flaky — see `artifacts/background-jobs.md`.

## 2. Testing Criteria

**Worth testing:**
- Usecases with branching/validation (REC/ORC/SAL business rules, e.g. `createUseCase.Execute`, `autoAdjustUseCase.Execute`)
- Repository methods that encode a query/constraint contract (`SumByUserAndCompetence`'s `COALESCE`, `ExistsByOriginAndCompetence`, unique index behavior)
- `TransactionClient`'s HTTP contract: timeout mapping, unexpected-status mapping, JSON shape
- Entity `BeforeCreate` hooks (UUID generation) and DB-level constraints (unique indexes, soft delete)
- Scheduler dedupe-by-competence logic and per-user error isolation (one failing user must not block the others)
- `balance.get`'s parallel aggregation via `errgroup` — partial-failure behavior (any source failing must return an error, never partial data — SAL's "no partial data" rule)
- Handler error mapping (`handleError` switch → HTTP status + error code + `BusinessErrorsTotal`)

**NOT worth testing:**
- GORM's own query execution, Gin's own routing/binding, Viper's own env parsing — framework behavior
- `cmd/server/main.go` wiring — a wiring/composition-root test only proves constructors were called
- Getters or structs with no logic
- `gofmt`/`go vet`/`golangci-lint`-catchable issues (unused vars, ineffective assignment) — that's `commands.lint`'s job, not a test's

## 3. Feature Implementation Checklist

When implementing a new feature (e.g. the upcoming MET/`reduction_goal` domain), use this checklist. For each artifact you create or modify, check the required test layers.

| Artifact created | Required tests | Guide |
|---|---|---|
| UseCase, no system boundary (`usecase/<domain>/<op>.go`) | Unit: branches + mocked repo | `artifacts/usecases.md` |
| UseCase calling `TransactionClient` | Unit: branches + mocked client (HTTP contract already covered by `artifacts/http-clients.md`) | `artifacts/usecases.md` |
| Repository (`repository/<entity>.go`) | Integration: real Postgres via testcontainers | `artifacts/repositories.md` |
| Entity (`entity/<name>.go`) | Unit: `BeforeCreate` + Integration: constraints/soft delete | `artifacts/entities.md` |
| Handler (`handler/<domain>/*.go`) | E2E: status codes + error codes, via `httptest` + real router | `artifacts/handlers.md` |
| New endpoint on `httpclient.TransactionClient` | Integration: fake `httptest.NewServer` (success/timeout/bad-status) | `artifacts/http-clients.md` |
| Background job change (`internal/scheduler`) | Unit: fakes + dedupe/error-isolation scenarios | `artifacts/background-jobs.md` |
| New middleware | E2E via router; integration if internal logic is complex (multi-step) | `artifacts/middleware.md` |

**How to use:** After implementing a feature, walk through each row. For each artifact you created or modified, read the corresponding guide and verify the tests exist. If a row doesn't apply, skip it.

## 4. Artifact Type Testing Guide

| Artifact Type | Pattern | Test Layer(s) | Guide |
|---|---|---|---|
| Use Cases | `usecase/<domain>/<operation>.go` | Unit | `artifacts/usecases.md` |
| Repositories | `repository/<entity>.go` | Integration (real DB) | `artifacts/repositories.md` |
| Entities | `entity/<name>.go` | Unit + Integration | `artifacts/entities.md` |
| Handlers | `handler/<domain>/*.go` | E2E only | `artifacts/handlers.md` |
| HTTP Clients | `pkg/httpclient/*.go` | Integration (fake server) | `artifacts/http-clients.md` |
| Background Jobs | `internal/scheduler/*.go` | Unit (fakes) | `artifacts/background-jobs.md` |
| Middleware | `pkg/observability/middleware.go` | E2E (+ integration if complex) | `artifacts/middleware.md` |
| Response Envelope | `pkg/response/*.go` | Unit | `artifacts/response-envelope.md` |
| Config | `src/config/config.go` | Unit (minimal) | `artifacts/config.md` |
| Future types | ReductionGoal (MET), cross-service authmiddleware | — | `artifacts/future-types.md` |

## 5. Anti-patterns — Do NOT Do This

- ❌ **Unit test repositories with a mocked `*gorm.DB`** — repository methods have no branching; only an integration test against real Postgres proves the query/constraint is correct (see `artifacts/repositories.md`)
- ❌ **Re-assert the HTTP contract inside every usecase that calls `TransactionClient`** — that contract has its own tests in `pkg/httpclient`; usecase tests should mock the client and focus on their own branching (§1)
- ❌ **Unit test handlers** — handlers are thin parse-and-delegate layers; the one piece of real logic (`handleError`) is exercised through E2E, not isolated (see `artifacts/handlers.md`)
- ❌ **Mock `*slog.Logger` or `*observability.ServiceMetrics`** — use the real constructors with a test namespace/name, as `router_test.go` and `scheduler_test.go` already do (§1)
- ❌ **Add a unique index on a soft-deleted entity without a partial-index test** — this project already paid for this bug once (`ADR-001`); see `references/gotchas.md`
- ❌ **Return partial data from `balance.get` when one of the 4 parallel sources fails** — SAL's contract is all-or-nothing; a test must assert the whole call fails, not that 3-of-4 fields are populated (`artifacts/usecases.md`)
- ❌ **Write a mirror test** — an assertion that just repeats the implementation's return value proves nothing (§2)
- ❌ **Chase a numeric coverage target** — this project uses pragmatic coverage: business-critical paths and system boundaries, not a percentage gate (`references/file-conventions.md`)

## 6. E2E Terminology Note

This guide uses "E2E" to mean HTTP-layer tests using `net/http/httptest` + the real Gin router (`router.ServeHTTP`) — i.e. what `router_test.go`'s `TestHealth_ReturnsOK` already does for `/health`. It does NOT mean browser-based or multi-service (hf-income-service ↔ hf-transaction-service) end-to-end tests; those are out of scope for this guide.

## 7. References

| Topic | File |
|---|---|
| External system mock strategies (Postgres, hf-transaction-service) | `references/external-systems.md` |
| Mock health rules & boundary principle | `references/mock-health-rules.md` |
| File naming, directory structure, coverage philosophy | `references/file-conventions.md` |
| Stack-specific gotchas & pitfalls | `references/gotchas.md` |

## 8. How to Use This Guide

This guide is organized as a multi-file skill:
- **This file (SKILL.md)** — always loaded. Contains core rules, quick reference, and anti-patterns.
- **`artifacts/`** — one file per artifact type. Read the relevant file when creating or modifying that type.
- **`references/`** — supporting content. Read when you need details on mock strategies, file conventions, or gotchas.

When working on a feature:
1. Check §3 (Feature Implementation Checklist) to identify which artifacts need tests
2. Read the corresponding `artifacts/*.md` file for the complete testing recipe
3. Consult `references/` files as needed for mock strategies, conventions, or pitfalls
