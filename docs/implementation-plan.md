# Plano de Implementação — hf-income-service

Data: 2026-06-10
FDDs considerados: fdd-001-rec, fdd-002-orc, fdd-003-met, fdd-004-sal
Estimativa total: ~24 dias

---

## Resumo de fases

| Fase | Domínio | Tasks | Estimativa |
|------|---------|-------|-----------|
| 0 | Infraestrutura base | T01–T04 | ~4d |
| 1 | REC — Receitas | T05–T08 | ~6d |
| 2 | ORC — Orçamento Global | T09–T12 | ~5d |
| 3 | MET — Metas de Redução | T13–T16 | ~6d |
| 4 | SAL — Saldo Mensal | T17–T18 | ~2.5d |
| 5 | Wiring e validação | T19 | ~1d |

---

## Fase 0 — Infraestrutura base

> Nenhum código de negócio. Apenas fundações que todas as fases dependem.

---

### T01 — Pacotes de suporte: config, database, response
**Estimativa:** M
**Depende de:** nada
**Arquivos:**
- `config/config.go` — carregamento de variáveis via Viper
- `pkg/database/postgres.go` — conexão GORM + AutoMigrate
- `pkg/response/response.go` — `response.Success()` e `response.Error()`

**Critérios de aceite:**
- [ ] `config.Load()` lê todas as variáveis de `.env.example` sem erro
- [ ] `database.Connect()` abre conexão GORM com PostgreSQL
- [ ] `response.Success(c, 200, data)` e `response.Error(c, 400, code, msg)` retornam JSON padronizado
- [ ] Projeto compila sem erro

**Risco:** baixo

---

### T02 — Entidades GORM e interfaces de repositório
**Estimativa:** M
**Depende de:** T01
**Arquivos:**
- `internal/entity/income.go`
- `internal/entity/global_budget.go`
- `internal/entity/reduction_goal.go`
- `internal/repository/interfaces.go`
- `ENTITIES.md` (verificar e atualizar se necessário)

**Critérios de aceite:**
- [ ] Entidades compilam com GORM tags corretas
- [ ] UUID gerado em `BeforeCreate` para todas as entidades
- [ ] Soft delete com `gorm.DeletedAt` em todas as entidades
- [ ] Interfaces `IncomeRepository`, `GlobalBudgetRepository`, `ReductionGoalRepository` definidas em `interfaces.go`
- [ ] `gorm.AutoMigrate` cria as três tabelas no PostgreSQL
- [ ] Índices únicos documentados nos FDDs incluídos nas entidades
- [ ] Projeto compila

**Risco:** baixo

---

### T03 — pkg/httpclient: cliente HTTP para hf-transaction-service
**Estimativa:** M
**Depende de:** T01
**Arquivos:**
- `pkg/httpclient/transaction_client.go`
- `pkg/httpclient/transaction_client_test.go`

**Critérios de aceite:**
- [ ] Interface `TransactionClient` com métodos: `GetExpenseTotals(ctx, userID, competence)`, `GetAccountsPayable(ctx, userID, dueDateUntil)`, `GetExpensesByCategory(ctx, userID, categoryID, competence)`
- [ ] Timeout de 5s configurado por chamada
- [ ] Retorna `ErrUpstreamTimeout` em timeout, `ErrUpstreamError` em status inesperado
- [ ] `lastDayOfMonth(competence)` implementado e testado: fevereiro, ano bissexto (2024-02 → 2024-02-29), dezembro
- [ ] Testes unitários com mock do servidor HTTP (`httptest.NewServer`)

**Risco:** médio (contrato do hf-transaction-service deve estar de pé para testes de integração futuros)

---

### T04 — Servidor Gin base: router, middleware, rotas vazias
**Estimativa:** P
**Depende de:** T01, T02, T03
**Arquivos:**
- `internal/handler/routes.go` — estrutura de rotas sob `/api/v1/` (endpoints podem retornar 501 temporariamente)
- `cmd/server/main.go` — inicialização parcial (sem use cases ainda)

**Critérios de aceite:**
- [ ] Servidor sobe na porta `APP_PORT` sem erro
- [ ] `GET /health` retorna `200`
- [ ] Middleware de observabilidade (`RequestMiddleware`) aplicado globalmente
- [ ] Métricas Prometheus expostas em `APP_METRICS_PORT/metrics`

**Risco:** baixo

> **Checkpoint após Fase 0**
> Antes de continuar:
> - [ ] `go build ./...` sem erro
> - [ ] Servidor sobe e `/health` responde
> - [ ] AutoMigrate cria as 3 tabelas no PostgreSQL

---

## Fase 1 — REC (Receitas)

---

### T05 — IncomeRepository: implementação + testes de integração
**Estimativa:** M
**Depende de:** T02, T04
**Arquivos:**
- `internal/repository/income.go`
- `internal/repository/income_test.go`

**Métodos:**
- `Create(ctx, income)`
- `GetByID(ctx, id)`
- `ListByUserAndCompetence(ctx, userID, competence)` — retorna itens + SUM para `total_income`
- `SumByUserAndCompetence(ctx, userID, competence)` — usado pelo SAL
- `Update(ctx, income)`
- `Delete(ctx, id)` — soft delete
- `ExistsByOriginAndCompetence(ctx, originID, competence)` — idempotência de propagação

**Critérios de aceite:**
- [ ] Todos os métodos cobertos por testes de integração com PostgreSQL real via `testcontainers-go`
- [ ] Soft delete: registro deletado não aparece em listagens
- [ ] `ExistsByOriginAndCompetence` retorna `true` para par já existente

**Risco:** baixo

---

### T06 — income use cases: create, get, list, delete + unit tests
**Estimativa:** G
**Depende de:** T05
**Arquivos:**
- `internal/usecase/income/create.go` + `create_test.go`
- `internal/usecase/income/get.go` + `get_test.go`
- `internal/usecase/income/delete.go` + `delete_test.go`

**Critérios de aceite:**
- [ ] `create`: valida REC-01 (campos obrigatórios), REC-02 (amount > 0), type válido, competence YYYY-MM
- [ ] `create`: `ErrMissingRequiredField`, `ErrInvalidAmount`, `ErrInvalidIncomeType`, `ErrInvalidCompetence`
- [ ] `get by ID`: retorna `ErrIncomeNotFound` para ID inexistente ou soft-deleted
- [ ] `list`: retorna itens + `total_income` agregado
- [ ] `delete`: soft delete; `ErrIncomeNotFound` se não existir
- [ ] Log INFO na entrada e saída de cada use case
- [ ] `BusinessErrorsTotal` incrementado em cada erro de regra
- [ ] Testes unitários com mock de `IncomeRepository`

**Risco:** baixo

---

### T07 — income use cases: update + propagate + unit tests
**Estimativa:** G
**Depende de:** T05
**Arquivos:**
- `internal/usecase/income/update.go` + `update_test.go`
- `internal/usecase/income/propagate.go` + `propagate_test.go`

**Critérios de aceite:**
- [ ] `update`: retorna `ErrIncomeNotFound` para ID inexistente
- [ ] `update`: retorna `ErrCannotEditPropagatedIncome` se `origin_id != nil` (REC-03)
- [ ] `update`: campos imutáveis (`user_id`, `origin_id`, `competence`) não alterados
- [ ] `propagate`: busca receitas `recurrent: true` da competência anterior
- [ ] `propagate`: idempotente — não cria duplicata se `(origin_id, competence)` já existe
- [ ] `propagate`: retorna contagem de registros propagados
- [ ] Testes unitários com mock cobrem idempotência e proteção de `origin_id`

**Risco:** baixo

---

### T08 — income handler + rotas
**Estimativa:** M
**Depende de:** T06, T07
**Arquivos:**
- `internal/handler/income/handler.go`
- Atualização de `internal/handler/routes.go`

**Critérios de aceite:**
- [ ] `POST /api/v1/income` → 201 / 400
- [ ] `GET /api/v1/income?user_id=&competence=` → 200
- [ ] `GET /api/v1/income/:id` → 200 / 404
- [ ] `PUT /api/v1/income/:id` → 200 / 400 / 404
- [ ] `DELETE /api/v1/income/:id` → 204 / 404
- [ ] `POST /api/v1/income/propagate` → 200 / 400
- [ ] Erros mapeados para `response.Error()` com codes corretos

**Risco:** baixo

> **Checkpoint após Fase 1**
> - [ ] Todos os endpoints REC respondem conforme FDD
> - [ ] `go test ./internal/usecase/income/... ./internal/repository/income_test.go` passa
> - [ ] `BusinessErrorsTotal{domain="REC"}` visível no `/metrics`

---

## Fase 2 — ORC (Orçamento Global)

---

### T09 — GlobalBudgetRepository: implementação + testes de integração
**Estimativa:** M
**Depende de:** T02, T04
**Arquivos:**
- `internal/repository/global_budget.go`
- `internal/repository/global_budget_test.go`

**Métodos:**
- `Create(ctx, budget)`
- `GetByUserAndCompetence(ctx, userID, competence)`
- `Update(ctx, budget)`
- `ExistsByUserAndCompetence(ctx, userID, competence)`

**Critérios de aceite:**
- [ ] Constraint única `(user_id, competence)` respeitada — segunda inserção retorna erro GORM
- [ ] Testes de integração com PostgreSQL real via `testcontainers-go`

**Risco:** baixo

---

### T10 — global_budget use cases: create, get, update + unit tests
**Estimativa:** M
**Depende de:** T09
**Arquivos:**
- `internal/usecase/global_budget/create.go` + `create_test.go`
- `internal/usecase/global_budget/get.go` + `get_test.go`
- `internal/usecase/global_budget/update.go` + `update_test.go`

**Critérios de aceite:**
- [ ] `create`: valida ORC-01 (campos obrigatórios, ceiling > 0), competence YYYY-MM
- [ ] `create`: retorna `ErrBudgetAlreadyExists` (409) se já existe para `user_id + competence`
- [ ] `update`: sempre força `auto_adjusted: false` (ORC-05)
- [ ] `update`: retorna `ErrBudgetNotFound` se não existir
- [ ] `BusinessErrorsTotal` e log INFO/WARN conforme FDD
- [ ] Testes unitários com mock de `GlobalBudgetRepository`

**Risco:** baixo

---

### T11 — global_budget use cases: auto_adjust + preview_next + unit tests
**Estimativa:** G
**Depende de:** T09, T03
**Arquivos:**
- `internal/usecase/global_budget/auto_adjust.go` + `auto_adjust_test.go`
- `internal/usecase/global_budget/preview_next.go` + `preview_next_test.go`

**Critérios de aceite:**
- [ ] `auto_adjust`: busca teto da competência anterior; retorna `ErrNoPreviousBudget` se não existe
- [ ] `auto_adjust`: ORC-03 — gasto < teto → novo teto = gasto, `auto_adjusted: true`
- [ ] `auto_adjust`: ORC-04 — gasto >= teto → teto permanece, `auto_adjusted: false`
- [ ] `auto_adjust`: upsert — atualiza se competência destino já existe, cria se não
- [ ] `auto_adjust`: retorna `ErrUpstreamUnavailable` (503) quando httpclient falha
- [ ] `auto_adjust`: log ERROR com `trace_id`, `upstream`, `status_code` em falha
- [ ] `preview_next`: mesma lógica sem persistir
- [ ] Testes unitários: ORC-03, ORC-04, falha upstream, `ErrNoPreviousBudget`
- [ ] `BusinessErrorsTotal{domain="ORC", rule_id="upstream"}` incrementado em falha

**Risco:** médio (depende do httpclient funcional)

---

### T12 — global_budget handler + rotas
**Estimativa:** M
**Depende de:** T10, T11
**Arquivos:**
- `internal/handler/global_budget/handler.go`
- Atualização de `internal/handler/routes.go`

**Critérios de aceite:**
- [ ] `POST /api/v1/budgets/global` → 201 / 400 / 409
- [ ] `GET /api/v1/budgets/global?user_id=&competence=` → 200 / 404
- [ ] `PUT /api/v1/budgets/global/:id` → 200 / 400 / 404
- [ ] `GET /api/v1/budgets/global/preview-next` → 200 / 400 / 503
- [ ] `POST /api/v1/budgets/global/auto-adjust` → 200 / 400 / 503

**Risco:** baixo

> **Checkpoint após Fase 2**
> - [ ] Todos os endpoints ORC respondem conforme FDD
> - [ ] `go test ./internal/usecase/global_budget/...` passa
> - [ ] ORC-03 e ORC-04 verificados manualmente ou por teste de integração leve

---

## Fase 3 — MET (Metas de Redução)

---

### T13 — ReductionGoalRepository: implementação + testes de integração
**Estimativa:** M
**Depende de:** T02, T04
**Arquivos:**
- `internal/repository/reduction_goal.go`
- `internal/repository/reduction_goal_test.go`

**Métodos:**
- `Create(ctx, goal)`
- `GetByID(ctx, id)`
- `ListByUserAndCompetence(ctx, userID, competence)`
- `Update(ctx, goal)`
- `Delete(ctx, id)` — soft delete
- `ListPendingClose(ctx, competence)` — metas com `achieved: null` para close-month
- `ExistsByUserCategoryAndCompetence(ctx, userID, categoryID, competence)`

**Critérios de aceite:**
- [ ] Constraint única `(user_id, category_id, competence)` respeitada
- [ ] Soft delete não aparece em listagens
- [ ] Testes de integração com PostgreSQL real via `testcontainers-go`

**Risco:** baixo

---

### T14 — reduction_goal use cases: create, list, update, delete + unit tests
**Estimativa:** G
**Depende de:** T13, T03
**Arquivos:**
- `internal/usecase/reduction_goal/create.go` + `create_test.go`
- `internal/usecase/reduction_goal/get.go` + `get_test.go`
- `internal/usecase/reduction_goal/update.go` + `update_test.go`
- `internal/usecase/reduction_goal/delete.go` + `delete_test.go`

**Critérios de aceite:**
- [ ] `create`: valida MET-04 (campos obrigatórios, target_amount > 0)
- [ ] `create`: retorna `ErrGoalAlreadyExists` (409) se já existe para `(user_id, category_id, competence)`
- [ ] `create`: tenta buscar `previous_amount` via httpclient; em falha cria com `previous_amount: null` + log WARN
- [ ] `update`: apenas `target_amount` editável; retorna `ErrGoalNotFound` se não existir
- [ ] `delete`: soft delete; `ErrGoalNotFound` se não existir
- [ ] Testes unitários: criação com httpclient disponível, criação com httpclient falhando (previous_amount: null)
- [ ] `BusinessErrorsTotal` incrementado nos erros de regra e upstream

**Risco:** médio (degradação graceful na criação)

---

### T15 — reduction_goal use cases: comparison + close_month + unit tests
**Estimativa:** G
**Depende de:** T13, T03
**Arquivos:**
- `internal/usecase/reduction_goal/comparison.go` + `comparison_test.go`
- `internal/usecase/reduction_goal/close_month.go` + `close_month_test.go`

**Critérios de aceite:**
- [ ] `comparison`: chamadas por categoria em paralelo com `errgroup`
- [ ] `comparison`: categoria com falha upstream retorna `current_month_amount: null` — request não falha (MET-05)
- [ ] `comparison`: preenche `previous_amount: null` retroativamente e persiste quando obtiver
- [ ] `comparison`: calcula `variation_pct`, `variation_label`, `on_track`, `target_progress_pct`
- [ ] `close_month`: seta `achieved: true` quando `current_month_amount <= target_amount` (MET-06)
- [ ] `close_month`: pula meta com falha upstream, loga ERROR com `goal_id`, continua demais
- [ ] Testes unitários: degradação parcial (1 categoria falha, outras ok), MET-06 verdadeiro e falso
- [ ] `BusinessErrorsTotal{domain="MET", rule_id="upstream"}` incrementado

**Risco:** médio (lógica de degradação parcial + paralelismo)

---

### T16 — reduction_goal handler + rotas
**Estimativa:** M
**Depende de:** T14, T15
**Arquivos:**
- `internal/handler/reduction_goal/handler.go`
- Atualização de `internal/handler/routes.go`

**Critérios de aceite:**
- [ ] `POST /api/v1/goals/reduction` → 201 / 400 / 409
- [ ] `GET /api/v1/goals/reduction?user_id=&competence=` → 200
- [ ] `PUT /api/v1/goals/reduction/:id` → 200 / 400 / 404
- [ ] `DELETE /api/v1/goals/reduction/:id` → 204 / 404
- [ ] `GET /api/v1/goals/reduction/comparison` → 200 (mesmo com degradação)
- [ ] `POST /api/v1/goals/reduction/close-month` → 200 / 400

**Risco:** baixo

> **Checkpoint após Fase 3**
> - [ ] Todos os endpoints MET respondem conforme FDD
> - [ ] Degradação parcial no comparativo verificada com mock
> - [ ] `go test ./internal/usecase/reduction_goal/...` passa

---

## Fase 4 — SAL (Saldo Mensal)

---

### T17 — balance use case: get + unit tests
**Estimativa:** G
**Depende de:** T05 (IncomeRepository), T09 (GlobalBudgetRepository), T03 (httpclient)
**Arquivos:**
- `internal/usecase/balance/get.go` + `get_test.go`

**Critérios de aceite:**
- [ ] Agrega 4 fontes em paralelo via `errgroup` (IncomeRepository, GlobalBudgetRepository, GetExpenseTotals, GetAccountsPayable)
- [ ] `balance_today = total_income - total_general` (SAL-01)
- [ ] `committed_bills = sum(pending accounts)` (SAL-02)
- [ ] `projected_balance = balance_today - committed_bills` (SAL-03)
- [ ] `is_projected_negative = projected_balance < 0` (SAL-05)
- [ ] `ceiling_usage_pct` como inteiro (ex: 64), `null` quando sem teto
- [ ] `total_personal` e `total_shared` presentes separadamente no response
- [ ] `total_income = 0` retorna 200 normalmente
- [ ] Falha em qualquer httpclient retorna 503 ou 502 — sem dados parciais
- [ ] Sub-spans OTEL: `balance.get.expense_totals` e `balance.get.accounts_payable`
- [ ] Testes: cenário completo, sem teto, total_income=0, falha GetExpenseTotals, falha GetAccountsPayable
- [ ] `BusinessErrorsTotal{domain="SAL", rule_id="upstream"}` incrementado em falha

**Risco:** médio (agrega mais fontes que qualquer outro use case)

---

### T18 — balance handler + rota
**Estimativa:** P
**Depende de:** T17
**Arquivos:**
- `internal/handler/balance/handler.go`
- Atualização de `internal/handler/routes.go`

**Critérios de aceite:**
- [ ] `GET /api/v1/balance?user_id=&competence=` → 200 / 400 / 502 / 503
- [ ] Shape do response conforme FDD-004 (todos os campos presentes)

**Risco:** baixo

> **Checkpoint após Fase 4**
> - [ ] `GET /balance` retorna response completo com servidor real
> - [ ] `go test ./internal/usecase/balance/...` passa
> - [ ] Traces com sub-spans visíveis no Grafana Tempo (validação opcional)

---

## Fase 5 — Wiring e validação final

---

### T19 — cmd/server/main.go: injeção de dependências + smoke tests
**Estimativa:** M
**Depende de:** T04, T08, T12, T16, T18
**Arquivos:**
- `cmd/server/main.go` — wiring completo de todas as dependências
- `.gitignore` — adicionar `task-brief.yaml`, `.env`

**Critérios de aceite:**
- [ ] Servidor sobe com todas as rotas registradas
- [ ] `go build ./cmd/server` sem erro
- [ ] `GET /health` → 200
- [ ] `POST /api/v1/income` cria e persiste com banco real
- [ ] `GET /api/v1/balance` retorna saldo calculado com banco real + hf-transaction-service em pé
- [ ] Métricas Prometheus acessíveis em `APP_METRICS_PORT/metrics`
- [ ] Log JSON em `APP_ENV=production`

**Risco:** baixo (apenas wiring — lógica já testada)

---

## Riscos consolidados

| Task | Risco | Motivo | Mitigação |
|------|-------|--------|-----------|
| T03 | médio | Contrato do hf-transaction-service pode divergir | Testes com `httptest.NewServer`; INTEGRATIONS.md como referência |
| T11 | médio | Auto-adjust depende do httpclient funcional | Mocks no unit test; teste manual com hf-transaction-service em pé |
| T14 | médio | Degradação graceful na criação (previous_amount null) | Teste unitário explícito para o caminho de falha |
| T15 | médio | Paralelismo no comparativo + degradação parcial | `errgroup` com contexto cancelável; teste de degradação |
| T17 | médio | Agrega 4 fontes — maior chance de falha parcial | 5 cenários de teste unitário distintos |

---

## Próximos passos

Após aprovação deste plano:
1. Criar tickets Jira para cada task (ou agrupar P em tickets M)
2. Iniciar pela Fase 0 — T01
3. Usar `/task <TICKET-ID>` para executar cada task com o workflow completo
