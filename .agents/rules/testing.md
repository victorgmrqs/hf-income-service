---
paths:
  - "src/**/*_test.go"
---
# Testing

- Name tests `Test<UseCase>_<Scenario>` (e.g. `TestCreateIncome_Success`, `TestCreateIncome_MissingRequiredField`, `TestCreateIncome_InvalidAmount`). The scenario suffix must match the mandatory matrix from the ticket/FDD: `_Success`, `_MissingRequiredField`/`_Invalid<Field>`, `_NotFound` (update/delete), `_UpstreamError` (when the usecase calls `pkg/httpclient`).
- Unit tests for usecases mock the repository/client interface with a **hand-written struct of function fields** (`mockIncomeRepository{CreateFn: func(...) error {...}}`), not a mocking framework. One `mock_repository.go` (and `mock_transaction_client.go` where relevant) per usecase package.
- Never mock `hf-transaction-service`'s `httpclient.TransactionClient` in a way that bypasses the interface — always through the same `mockXFn` pattern, so the real client stays swappable.
- Repository tests are integration tests against a real PostgreSQL via `testcontainers-go`, in files suffixed `_integration_test.go` (separate from unit tests, not run by default `go test ./...` unless the container is available — see CI job).
- Build inputs with a shared `valid<X>Input()` helper per test file and mutate only the field under test — keeps each scenario a one-line diff from the happy path.
- Use `testLogger()` (a no-op/discard `*slog.Logger`) instead of `slog.Default()` in unit tests — never assert on log output unless the test is specifically about logging.
- Assert behavior, not implementation: check the returned `Output`/error, not that a specific internal method was called, unless the scenario is explicitly "repo must not be called" (validation short-circuit) — use `t.Fatal` inside the mock function for that case.

Example (real, `create_test.go`):

```go
repo := &mockIncomeRepository{
    CreateFn: func(_ context.Context, income *entity.Income) error {
        income.ID = uuid.New()
        return nil
    },
}
uc := NewCreateUseCase(repo, testLogger())
```
