> Part of the testing-guide-hf-income-service skill (see ../SKILL.md).

# Mock Health Rules

## The boundary rule, in this project's terms

Mock `repository.<Entity>Repository` and `httpclient.TransactionClient` interfaces in usecase unit tests — they're the architecturally significant boundary, and each has its own dedicated test suite proving its real contract (`artifacts/repositories.md`, `artifacts/http-clients.md`). Never mock `*slog.Logger` or `*observability.ServiceMetrics` — construct real instances (`observability.NewLogger("test")`, `observability.NewServiceMetrics("<namespace>")`); they're configured libs, and a mock would never catch a wrong `slog.String` key or a metric label typo.

If a usecase test needs more than 2-3 mocked methods to set up, that's a signal the usecase is doing too much — consider whether some of that logic belongs in a different layer, per the fundamentals' mock-health litmus test ("can you describe the observable behavior without referencing mock interactions?").

## How mocks are written in this project

Hand-written structs with exported `XxxFn func(...)` fields — **not** a mocking framework (no `gomock`, no `testify/mock`; `testify` is only an indirect transitive dependency, not used anywhere in this codebase):

```go
type mockIncomeRepository struct {
	CreateFn                     func(ctx context.Context, income *entity.Income) error
	FindByIDFn                   func(ctx context.Context, id uuid.UUID) (*entity.Income, error)
	// ...only the methods a given test file actually exercises
}

func (m *mockIncomeRepository) Create(ctx context.Context, income *entity.Income) error {
	return m.CreateFn(ctx, income)
}
// each interface method delegates to its Fn field
```

- One `mock_repository.go` (and `mock_transaction_client.go` where relevant) **per usecase package** — don't share a mock across `income` and `global_budget`, even though both mock similarly-shaped interfaces.
- Leave a `Fn` field `nil` if a scenario should never call that method — calling a `nil` func panics, which is the desired failure mode for "this method must not be invoked" assertions (see `TestCreateIncome_MissingRequiredField`'s pattern of `t.Fatal` inside the `Fn` body instead, when the interface only has one method to guard).
- Field names are exported (`CreateFn`, not `createFn`) — this is a divergence from the (now-removed) previous testing-guide skill and should be treated as the canonical convention going forward.

## What to mock vs use real, summarized

| Dependency kind | In this project | Treatment |
|---|---|---|
| Owned repository interface | `IncomeRepository`, `GlobalBudgetRepository`, `ReductionGoalRepository` | Mock in usecase unit tests |
| Owned HTTP client interface | `httpclient.TransactionClient` | Mock in usecase unit tests |
| Configured lib | `*slog.Logger`, `*observability.ServiceMetrics` | Real instance, test config/namespace |
| Side-effect dependency | none in this service today | — |
