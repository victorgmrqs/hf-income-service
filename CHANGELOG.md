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
- Entidades GORM `GlobalBudget` (ORC) e `ReductionGoal` (MET) em `src/internal/entity/`; sem soft delete em `GlobalBudget` (substituição direta — ORC-01), soft delete em `ReductionGoal` (MET-04); índices únicos `(user_id, competence)` e `(user_id, category_id, competence)` criados via AutoMigrate; interfaces `GlobalBudgetRepository` e `ReductionGoalRepository` adicionadas a `src/internal/repository/interfaces.go`. (HF-59)
- Propagação de receitas recorrentes (REC-04): `POST /api/v1/income/propagate` idempotente; use case `propagate` + repo `ListRecurrentByCompetence`/`ExistsByOriginAndCompetence`; índice único `(origin_id, competence)` garante a idempotência no banco. (HF-98)
- CRUD de Teto Global (ORC): `GlobalBudgetRepository` (implementação GORM), use cases create/get/update e handler em `/api/v1/budgets/global` (`POST`, `GET ?user_id=&competence=`, `PUT /:id`); ORC-01 — unicidade `(user_id, competence)` com `409 BUDGET_ALREADY_EXISTS`, `ceiling > 0` e `competence` `YYYY-MM`; edição manual força `auto_adjusted=false` (ORC-05); `BusinessErrorsTotal{ORC,ORC-01}` nas violações e `AutoMigrate` da tabela `global_budgets`. (HF-43)
- Auto-ajuste progressivo do teto global (ORC-03/04/05): `POST /api/v1/budgets/global/auto-adjust` (idempotente via upsert) e `GET /api/v1/budgets/global/preview-next` (cálculo sem persistir, `adjustment_reason`); consome o gasto do mês anterior via `httpclient.GetExpenseTotals` (hf-transaction-service) — falha upstream retorna `503 UPSTREAM_UNAVAILABLE` com log ERROR (`trace_id`, `upstream`); sem teto anterior retorna `400 NO_PREVIOUS_BUDGET` (ORC-03); métricas `BusinessErrorsTotal{ORC,ORC-03}` e `{ORC,upstream}`; repositório ganha `ExistsByUserAndCompetence`, `Upsert` e `Delete` (soft). Absorve HF-64/66/67. (HF-44)

- Saldo mensal (SAL): `GET /api/v1/balance?user_id=&competence=` — agrega 4 fontes em paralelo via `errgroup` (receitas e teto locais + `GetExpenseTotals`/`GetAccountsPayable` do hf-transaction-service) sem persistência e sem dados parciais; SAL-01/02/03/04/05 e ORC-06/07 (`ceiling_exceeded`, `ceiling_usage_pct` inteiro, `null` sem teto); `503 UPSTREAM_TIMEOUT` / `502 UPSTREAM_ERROR`; spans OTEL `balance.get` + sub-spans por chamada; `IncomeRepository` ganha `SumByUserAndCompetence` (pendência do HF-60). Absorve HF-72/74. (HF-41)
- Scheduler interno (`src/internal/scheduler`): goroutine com ticker de 1h que, no 1º dia do mês (`SCHEDULER_TZ`, default `America/Sao_Paulo`), dispara `income.PropagateUseCase` (REC-04, competência corrente) e, para cada usuário com teto na competência anterior (`GlobalBudgetRepository.ListUserIDsByCompetence`, novo), `global_budget.AutoAdjustUseCase` (ORC-05); falha por job/usuário loga ERROR com `trace_id` e não interrompe o ciclo; dedupe por competência evita repetição no mesmo dia; encerramento gracioso via `signal.NotifyContext`; `SCHEDULER_ENABLED` (default `true`) desativa a goroutine; TZ inválida cai para UTC com log WARN. (HF-38)

### Changed
- `GlobalBudget` passa a usar soft delete (`deleted_at`) — supersede a decisão "sem soft delete" do HF-59 (ver `docs/adr/ADR-001`); a unicidade ORC-01 agora é índice único **parcial** `(user_id, competence) WHERE deleted_at IS NULL` (`repository.EnsureGlobalBudgetIndexes`), permitindo recriar o teto após exclusão. Timeout do `pkg/httpclient` reduzido de 10s para 5s (INTEGRATIONS.md/FDD-002 §6); contrato do `auto-adjust` no `openapi.yaml` alinhado ao FDD (`competence` de destino, resposta `200`). (HF-44)
- Reestruturação de pastas: `internal/`, `config/`, `pkg/`, `cmd/` movidos para `src/`; `go.mod`/`go.sum` permanecem na raiz; sem mudança de regra de negócio. (HF-97)

### Fixed
- `pkg/httpclient` consolidado: os merges de HF-57 e HF-44 redeclararam `TransactionClient`/`ExpenseTotalsOutput` no mesmo pacote e `development` parou de compilar; o `client.go` do HF-44 foi removido em favor do `transaction_client.go` (cliente canônico), `main.go` migrado para `NewTransactionClient` e `GetExpenseTotals` ganhou guard contra `data:null`. (HF-41)
- `pkg/observability/tracing.go` reformatado com gofmt. (HF-91)

### Dependencies
- `golang.org/x/sync` v0.20.0 promovido de indireto para direto (`errgroup` no use case de saldo). (HF-41)
- `go mod tidy`: requires indiretos sincronizados no `go.mod` (entradas faltantes de `gin` e transitivos). (HF-91)
- Adicionados `github.com/spf13/viper`, `gorm.io/gorm`, `gorm.io/driver/postgres`. (HF-56)
- Adicionados `github.com/shopspring/decimal` (direto) e, para testes de integração, `github.com/testcontainers/testcontainers-go` (+ módulo postgres). (HF-37)
