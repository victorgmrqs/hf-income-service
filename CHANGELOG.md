# Changelog

Todas as mudanças relevantes deste serviço são documentadas aqui.
Formato baseado em [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/).
Cada entrada referencia o ticket Jira (`(HF-XX)`). Datas em `YYYY-MM-DD`.

## [Unreleased]

### Added
- Gates de qualidade no CI: job `quality` (gofmt + go vet + golangci-lint só código novo do PR) e job `docs-guard` (em pull_request); `build-and-push` passa a depender de `quality`. (HF-91)
- `.golangci.yml` (errcheck, govet, ineffassign, staticcheck, unused) e `scripts/docs-guard.sh` (código em `internal/` sem CHANGELOG/teste no mesmo diff falha o PR). (HF-91)
- Skills do workflow versionadas: `/task`, `/code-review-task`, `/docs-sync` em `.claude/skills/<nome>/SKILL.md` (Claude Code) e `.agents/skills/<nome>.md` (agy); `task.md` plano antigo removido. (HF-96)
- Pacotes de suporte: `src/config` (`Load()` via Viper), `src/pkg/database` (`Connect()` GORM/PostgreSQL) e `src/pkg/response` (`Success()`/`Error()` com envelope `{data,error}`). (HF-56)
- `TransactionClient` de alto nível em `src/pkg/httpclient/transaction_client.go`: interface tipada com `GetExpenseTotals`, `GetAccountsPayable` e `GetExpensesByCategory`; erros sentinela `ErrUpstreamTimeout`/`ErrUpstreamError`; timeout de 5s por chamada via `context.WithTimeout`; helper `LastDayOfMonth` exportado; INTEGRATIONS.md atualizado com Endpoint 3 placeholder (MET). (HF-57)
- Servidor HTTP base: `src/internal/handler/router.go` (Gin + middleware de observabilidade + `GET /health`), `src/cmd/server/main.go` (wiring config → observability → DB → server) e `src/pkg/httpclient` (cliente do hf-transaction-service via interface `TransactionClient`); `/metrics` Prometheus servido em `APP_METRICS_PORT`. (HF-36)
- CRUD de Receitas (REC): entidade `Income` (GORM, soft delete, `origin_id`), `IncomeRepository`, use cases create/get/list/update/delete e handler em `/api/v1/income` (com `total_income` agregado); validações REC-01/02/03 com `BusinessErrorsTotal` por `rule_id` e `AutoMigrate` da tabela `incomes`. (HF-37)

### Changed
- Reestruturação de pastas: `internal/`, `config/`, `pkg/`, `cmd/` movidos para `src/`; `go.mod`/`go.sum` permanecem na raiz; sem mudança de regra de negócio. (HF-97)

### Fixed
- `pkg/observability/tracing.go` reformatado com gofmt. (HF-91)

### Dependencies
- `go mod tidy`: requires indiretos sincronizados no `go.mod` (entradas faltantes de `gin` e transitivos). (HF-91)
- Adicionados `github.com/spf13/viper`, `gorm.io/gorm`, `gorm.io/driver/postgres`. (HF-56)
- Adicionados `github.com/shopspring/decimal` (direto) e, para testes de integração, `github.com/testcontainers/testcontainers-go` (+ módulo postgres). (HF-37)
