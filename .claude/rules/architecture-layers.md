---
paths:
  - "src/internal/**/*.go"
---
# Architecture Layers

Dependency direction is one-way: `Handler → UseCase → Repository (interface) → DB`.

- Business logic lives only in `usecase/<domain>/*.go`, one operation per file (`create.go`, `get.go`, `update.go`, `delete.go`, ...). Never put validation or rule logic in the handler.
- Each usecase file declares its own interface (`type CreateUseCase interface { Execute(ctx, input) (*Output, error) }`) plus a lowercase struct implementing it and a `NewXUseCase(repo, logger) XUseCase` constructor. Dependencies (repository, logger, other clients) are injected — never instantiated inside the usecase.
- Usecases depend on `repository.<Entity>Repository` interfaces (defined in `src/internal/repository/interfaces.go`), never on the concrete GORM implementation. This keeps unit tests mockable without a database.
- Handlers only: parse/validate the HTTP request into an `Input` struct, call the injected usecase interface, map the result/error to `response.Success`/`response.Error`. Handlers never touch `gorm.DB`, `entity.*` persistence tags, or repository implementations directly.
- Repository implementations (`src/internal/repository/<entity>.go`) are the only place allowed to import `gorm.io/gorm` for queries; they translate GORM errors (e.g. `gorm.ErrRecordNotFound`) as-is so usecases can `errors.Is` against them.
- Cross-service calls (to `hf-transaction-service`) go through `pkg/httpclient`, injected into usecases the same way as a repository — never called directly from a handler.

Example (real, `income/create.go`):

```go
type CreateUseCase interface {
    Execute(ctx context.Context, input CreateInput) (*IncomeOutput, error)
}

type createUseCase struct {
    repo   repository.IncomeRepository
    logger *slog.Logger
}

func NewCreateUseCase(repo repository.IncomeRepository, logger *slog.Logger) CreateUseCase {
    return &createUseCase{repo: repo, logger: logger}
}
```
