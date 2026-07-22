> Part of the testing-guide-hf-income-service skill (see ../SKILL.md).

# File Conventions

## Naming

- Unit tests (usecases, entities' `BeforeCreate`, response envelope): `Test<Subject>_<Scenario>`, e.g. `TestCreateIncome_Success`, `TestCreateIncome_InvalidAmount`, `TestIncome_BeforeCreate_GeneratesUUID`. `<Subject>` is the thing under test (often the usecase minus "UseCase"), not the file name.
- Repository/HTTP-client integration tests: `Test<Package/Client>_<Scenario>`, e.g. `TestGetExpenseTotals_Timeout_ReturnsUpstreamTimeout`. The `income`/`global_budget` repository tests currently use one larger `Test<Entity>Repository_Integration` per file rather than one test per scenario — either style is acceptable; prefer splitting into scenario-named tests going forward for clearer failure output, per the pattern already used in `pkg/httpclient`.
- Handler E2E tests (new — see `artifacts/handlers.md`): `Test<Domain>Handler_<Method>_<Scenario>`, e.g. `TestIncomeHandler_Create_InvalidAmount`.

## Directory placement

- Colocated, always — every `_test.go` file lives next to the code it tests, in the same directory. There is no centralized `test/` or `tests/` directory in this project.
- Package choice: usecase/entity unit tests use the **same package** as the code under test (`package income`, not `package income_test`) when they need access to unexported helpers (`validCreateInput()`, `testLogger()`); `pkg/httpclient` and `pkg/response` tests use the **external** `_test` package (`package httpclient_test`) since they only exercise the public API.
- Integration tests: same directory as unit tests, distinguished only by the `_integration_test.go` filename suffix — not a separate subdirectory.

## Configuration

- No test config file — `go test` runs directly off `go.mod` (`go 1.25.0`). No `moduleNameMapper`-equivalent needed (Go has no path-aliasing to configure).
- Commands (mirrors `.dev-workflow/workflow.config.yaml`):
  ```bash
  go test ./...              # all tests — repository/entity integration tests self-skip without Docker
  go test -cover ./...       # coverage summary
  go test -run TestCreateIncome_Success ./src/internal/usecase/income/   # single test
  ```
- CI (`ci-cd.yml`, `test` job) runs `go test ./...` with Docker available, so integration tests run for real there, not skipped.

## Coverage philosophy: pragmatic

No numeric coverage gate exists in CI (`quality_gates.coverage_threshold: null` in `workflow.config.yaml`) and none should be added as a blocking check. Coverage is a byproduct of following §3's Feature Implementation Checklist (every artifact type gets the tests its layer assignment calls for), not a percentage target to chase. The DoD's bar is the **mandatory scenario matrix** from the ticket/FDD (`_Success`, `_MissingRequiredField`, `_NotFound`, `_UpstreamError`, ...) — a change is done when those named scenarios exist and pass, not when `go test -cover` crosses some number. If `go test -cover ./...` shows an untested branch that isn't part of that matrix and isn't a system boundary, it's fine to leave it — see §2's "NOT worth testing" list before adding a test just to move the number.
