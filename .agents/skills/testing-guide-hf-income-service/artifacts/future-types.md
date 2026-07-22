> Part of the `testing-guide-hf-income-service` skill (see `../SKILL.md`).

# Future Types

Artifact types anticipated by the backlog (`docs/implementation-plan.md`) but not yet built. Apply the matching existing guide when they land — don't invent a new pattern.

## MET domain (`reduction_goal`) — next up (HF-68/71/69/70)

- **`repository/reduction_goal.go`** → follow `artifacts/repositories.md` exactly. The one thing to get right from day one: the composite unique index `(user_id, category_id, competence)` on a soft-deleted entity — write the delete-then-recreate integration test (see `references/gotchas.md`) in the same PR that adds the index, not as a follow-up.
- **`usecase/reduction_goal/*.go`** → follow `artifacts/usecases.md`. Two usecases cross a system boundary and need the same "mock the client, don't re-test its HTTP contract" treatment as `balance.get`:
  - `create` calls `TransactionClient` to fetch `previous_amount` and must degrade gracefully (persist with `previous_amount: null` + WARN log) on failure — test both the success and the degraded-creation path.
  - `comparison` fans out per-category calls in parallel (`errgroup`, like `balance.get`) but with a **different** partial-failure contract than SAL: a single category's upstream failure must NOT fail the whole request — it returns `current_month_amount: null` for that category only (MET-05). Test this explicitly; it's the opposite of SAL's all-or-nothing rule, and copying `balance.get`'s error-propagation pattern here would be a real bug.
  - `close_month` must skip (not abort) a goal whose upstream call fails, logging ERROR with `goal_id`, and continue processing the rest — same error-isolation shape as the scheduler (`artifacts/background-jobs.md`).
- **`handler/reduction_goal/*.go`** → follow `artifacts/handlers.md`.

## Cross-service auth middleware (HF-117, future — not in this repo yet)

`RNF-SEG-01` ("Autenticação JWT em todos os endpoints") is explicitly deferred past the MVP. When a shared `authmiddleware` package (from the future `hf-auth-service`) is wired into this service's routes:

- The middleware itself gets its own test suite in its origin package — don't duplicate that here.
- This service only needs an E2E test per protected route proving a request without a valid token is rejected before reaching the handler (`artifacts/middleware.md`'s pattern, extended with a 401 assertion) — same "accepted/rejected requests" layer assignment the fundamentals prescribe for middleware in general.
