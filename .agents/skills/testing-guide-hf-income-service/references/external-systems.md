> Part of the testing-guide-hf-income-service skill (see ../SKILL.md).

# External Systems

| System | Strategy | Why | How |
|---|---|---|---|
| **PostgreSQL** | Real (Docker via testcontainers-go) | Fast to spin up (<5s), fully controllable, the query/constraint contract genuinely matters — see `references/gotchas.md` for the soft-delete/unique-index interaction this project already hit | `testcontainers-go/modules/postgres`, image pinned to `postgres:16-alpine` (never `:latest`), `wait.ForListeningPort("5432/tcp")`. `t.Skip` (not `t.Fatal`) if Docker is unavailable locally — CI always has it. |
| **hf-transaction-service** (external HTTP API) | Fake (`httptest.NewServer`) | No network flakiness, no dependency on the sibling service being up, full control over timeout/error scenarios | `httptest.NewServer(http.HandlerFunc(...))`, inject the `*http.Client` (and its timeout) into `httpclient.NewTransactionClient` — never point tests at a real running `hf-transaction-service` |

No message queue, cache, email, or object storage in this service — nothing else to configure here.

## Setup/teardown

- **Postgres:** `t.Cleanup(func() { _ = container.Terminate(ctx) })` right after the container starts, before any assertion — guarantees cleanup even if the test fails or panics. `AutoMigrate` only the entities the specific test file needs.
- **hf-transaction-service fake:** `defer srv.Close()` right after `httptest.NewServer(...)` — a leaked server holds a real OS port for the life of the test binary.
