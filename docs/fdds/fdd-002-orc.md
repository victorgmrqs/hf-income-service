# FDD: Orcamento Global (ORC)

Versão: 1.0
Data: 2026-06-09
Domínio: ORC

---

### 1. Contexto e motivação técnica

O domínio `global_budget` define um teto mensal de gastos por `user_id + competence`. É independente do domínio REC em persistência, mas depende do `hf-transaction-service` para obter o total gasto no mês anterior durante o auto-ajuste.

Encaixe na arquitetura: fluxo `Handler > UseCase > Repository > PostgreSQL` para CRUD; `Handler > UseCase > HTTPClient > hf-transaction-service` para auto-ajuste e preview.

Relação com outros dominios:
- `balance/get.go` (SAL) consome `GlobalBudgetRepository` para calcular `ceiling_usage_pct` e `ceiling_exceeded`
- ORC nao depende de REC em persistencia — opera de forma independente

`user_id` e referencia externa sem FK local. Validacao de existencia do usuario sera responsabilidade do futuro middleware JWT.

---

### 2. Objetivos tecnicos

- CRUD do teto global: criar, consultar e editar por `user_id + competence`
- Auto-ajuste automatico via `POST /budgets/global/auto-adjust` — consome gasto do mes anterior do hf-transaction-service (ORC-03/04/05)
- Preview do auto-ajuste sem persistir via `GET /budgets/global/preview-next` (ORC-03/04)
- Flag `auto_adjusted: bool` auditavel em todos os registros (ORC-03/04/05)
- Latencia p95 < 300ms para leitura/escrita simples; < 1s para endpoints com chamada ao hf-transaction-service

**Invariantes que nao podem ser violados:**
- Cada usuario tem no maximo um teto por competencia (ORC-01) — constraint unica em `(user_id, competence)`
- `ceiling` nunca pode ser zero ou negativo
- Edicao manual sempre forca `auto_adjusted: false` (ORC-05)

---

### 3. Escopo e exclusoes

**Incluido**
- `POST /budgets/global` — criar teto para uma competencia
- `GET /budgets/global?user_id=&competence=` — consultar teto vigente
- `PUT /budgets/global/:id` — editar manualmente (forca `auto_adjusted: false`)
- `GET /budgets/global/preview-next?user_id=` — preview do auto-ajuste sem aplicar
- `POST /budgets/global/auto-adjust` — aplicar auto-ajuste (chamado por job no 1o do mes)
- Flag `auto_adjusted` auditavel em todos os registros

**Excluido**
- Orcamento por categoria — calculo fica no SAL/balance
- `ceiling_usage_pct` e `ceiling_exceeded` — campos calculados retornados pelo `/balance`
- Historico de ajustes (audit log de alteracoes no teto) — funcionalidade futura
- Validacao de existencia do `user_id` — responsabilidade do futuro auth service

---

### 4. Fluxos detalhados

**Fluxo principal: `POST /budgets/global`**
1. Handler valida campos obrigatorios: `user_id`, `competence`, `ceiling`
2. UseCase verifica se ja existe teto para `user_id + competence` — se sim, retorna `409`
3. Cria entidade `GlobalBudget` com `auto_adjusted: false`
4. Persiste e retorna `201`

**Fluxo principal: `PUT /budgets/global/:id`**
1. Handler extrai `id` da rota e `ceiling` do body
2. UseCase busca o registro — `404` se nao existir
3. Atualiza `ceiling` e forca `auto_adjusted: false` (ORC-05)
4. Retorna `200`

**Fluxo principal: `POST /budgets/global/auto-adjust`**
1. Handler recebe `{ "user_id": "uuid", "competence": "2026-07" }` (competencia de destino)
2. UseCase calcula competencia anterior (`2026-06`)
3. Busca teto da competencia anterior — se nao existir, retorna `400` com `ErrNoPreviousBudget`
4. Chama `httpclient.GetExpenseTotals(user_id, "2026-06")` para obter `total_general`
5. Se httpclient falhar → log ERROR com `trace_id`, `user_id`, `competence`, `status_code` upstream + retorna `503`
6. Aplica logica ORC-03/04:
   - Se `total_general < ceiling_anterior` → `novo_ceiling = total_general`, `auto_adjusted: true`
   - Se `total_general >= ceiling_anterior` → `novo_ceiling = ceiling_anterior`, `auto_adjusted: false`
7. Upsert: atualiza teto da competencia de destino se ja existe, cria se nao existe
8. Retorna `200` com o teto resultante

**Fluxo principal: `GET /budgets/global/preview-next`**
- Mesma logica do auto-adjust (passos 2-6), mas sem persistir — retorna apenas o calculo

**Fluxos alternativos**
- `GET /budgets/global` sem teto cadastrado para competencia: retorna `404`
- `POST /budgets/global/auto-adjust` sem receitas no mes anterior: `total_general = 0` → novo teto = 0 (registrar como caso de atencao no log WARN)
- Primeiro mes de uso sem teto anterior: `ErrNoPreviousBudget` → frontend orienta criacao manual

---

### 5. Contratos publicos

**`POST /budgets/global`**
- Rota: `POST /api/v1/budgets/global`
- Status codes: `201` criado, `400` validacao, `409` ja existe, `500` erro interno

Request:
```json
{
  "user_id": "uuid",
  "competence": "2026-06",
  "ceiling": "5000.00"
}
```

Response 201:
```json
{
  "data": {
    "id": "uuid",
    "user_id": "uuid",
    "competence": "2026-06",
    "ceiling": "5000.00",
    "auto_adjusted": false,
    "created_at": "2026-06-09T10:00:00Z"
  },
  "error": null
}
```

---

**`GET /budgets/global?user_id=&competence=`**
- Rota: `GET /api/v1/budgets/global`
- Status codes: `200` ok, `404` nao encontrado, `500` erro interno

Response 200:
```json
{
  "data": {
    "id": "uuid",
    "user_id": "uuid",
    "competence": "2026-06",
    "ceiling": "5000.00",
    "auto_adjusted": false,
    "created_at": "2026-06-09T10:00:00Z"
  },
  "error": null
}
```

---

**`PUT /budgets/global/:id`**
- Rota: `PUT /api/v1/budgets/global/:id`
- Status codes: `200` ok, `400` validacao, `404` nao encontrado, `500` erro interno

Request:
```json
{ "ceiling": "4800.00" }
```

---

**`GET /budgets/global/preview-next?user_id=`**
- Rota: `GET /api/v1/budgets/global/preview-next`
- Status codes: `200` ok, `400` sem teto anterior, `503` upstream indisponivel, `500` erro interno

Response 200:
```json
{
  "data": {
    "current_competence": "2026-06",
    "current_ceiling": "5000.00",
    "current_spending": "4200.00",
    "next_competence": "2026-07",
    "suggested_ceiling": "4200.00",
    "adjustment_reason": "spending_below_ceiling"
  },
  "error": null
}
```

---

**`POST /budgets/global/auto-adjust`**
- Rota: `POST /api/v1/budgets/global/auto-adjust`
- Status codes: `200` ok, `400` sem teto anterior, `503` upstream indisponivel, `500` erro interno

Request:
```json
{ "user_id": "uuid", "competence": "2026-07" }
```

Response 200:
```json
{
  "data": {
    "id": "uuid",
    "user_id": "uuid",
    "competence": "2026-07",
    "ceiling": "4200.00",
    "auto_adjusted": true
  },
  "error": null
}
```

Response 503:
```json
{
  "data": null,
  "error": { "code": "UPSTREAM_UNAVAILABLE", "message": "could not fetch expense data, please retry" }
}
```

---

### 6. Erros, excecoes e fallback

| Condicao | Variavel Go | Rule ID | HTTP | Code |
|----------|------------|---------|------|------|
| Campo obrigatorio ausente | `ErrMissingRequiredField` | ORC-01 | 400 | `MISSING_REQUIRED_FIELD` |
| `ceiling` <= 0 | `ErrInvalidCeiling` | ORC-01 | 400 | `INVALID_CEILING` |
| `competence` fora do formato `YYYY-MM` | `ErrInvalidCompetence` | ORC-01 | 400 | `INVALID_COMPETENCE` |
| Ja existe teto para `user_id + competence` | `ErrBudgetAlreadyExists` | ORC-01 | 409 | `BUDGET_ALREADY_EXISTS` |
| Teto nao encontrado | `ErrBudgetNotFound` | — | 404 | `BUDGET_NOT_FOUND` |
| Sem teto anterior para base do auto-adjust | `ErrNoPreviousBudget` | ORC-03 | 400 | `NO_PREVIOUS_BUDGET` |
| hf-transaction-service indisponivel | `ErrUpstreamUnavailable` | — | 503 | `UPSTREAM_UNAVAILABLE` |

**Resiliencia:**
- Timeout: 5s nas chamadas ao hf-transaction-service (configurado no `pkg/httpclient`)
- Falha upstream: log ERROR com `trace_id`, `upstream="hf-transaction-service"`, `status_code`, `user_id`, `competence`
- Nao ha fallback automatico no MVP — job pode ser re-executado manualmente
- Plano futuro: mensageria + alerta Slack/Telegram para notificacao de falha de integracao

---

### 7. Observabilidade

**Spans OTEL** (criados no use case)
- `global_budget.create`
- `global_budget.get`
- `global_budget.update`
- `global_budget.auto_adjust`
- `global_budget.preview_next`

**Metricas**
```go
// ORC-01 — campo invalido ou ceiling invalido
metrics.BusinessErrorsTotal.WithLabelValues("ORC", "ORC-01").Inc()
// ORC-03 — sem teto anterior
metrics.BusinessErrorsTotal.WithLabelValues("ORC", "ORC-03").Inc()
// upstream — hf-transaction-service indisponivel
metrics.BusinessErrorsTotal.WithLabelValues("ORC", "upstream").Inc()
```

**Campos de log obrigatorios**
- Entrada (INFO): `operation`, `user_id`, `competence`
- Saida (INFO): `operation`, `duration_ms`, `budget_id`, `auto_adjusted`
- Erro upstream (ERROR): `trace_id`, `upstream`, `status_code`, `user_id`, `competence`
- Warn/rule (WARN): `trace_id`, `rule_id`, `reason`

**Alertas minimos no Grafana**
- `business_errors_total{domain="ORC", rule_id="upstream"}` > 3 em 10 minutos — falha de integracao com hf-transaction-service

---

### 8. Dependencias e compatibilidade

| Componente | Versao minima | Observacoes |
|-----------|--------------|-------------|
| Go | 1.23 | |
| PostgreSQL | 16 | |
| GORM | v2 | |
| shopspring/decimal | v1 | Obrigatorio para `ceiling` |
| hf-transaction-service | — | Contrato documentado em INTEGRATIONS.md |

**Dependencias entre use cases**
- `balance/get.go` usa `GlobalBudgetRepository.GetByUserAndCompetence` para `ceiling_usage_pct` e `ceiling_exceeded`
- `global_budget/auto_adjust.go` chama `httpclient.GetExpenseTotals` — depende do contrato de INTEGRATIONS.md

**Indice recomendado**
```sql
CREATE UNIQUE INDEX idx_global_budgets_user_competence ON global_budgets (user_id, competence);
```

---

### 9. Criterios de aceite tecnicos

**Funcional**
- [ ] `POST /budgets/global` cria teto e retorna `201`
- [ ] `POST /budgets/global` retorna `409` se ja existe teto para `user_id + competence`
- [ ] `PUT /budgets/global/:id` sempre seta `auto_adjusted: false`
- [ ] `POST /budgets/global/auto-adjust` aplica ORC-03: gasto < teto => novo teto = gasto, `auto_adjusted: true`
- [ ] `POST /budgets/global/auto-adjust` aplica ORC-04: gasto >= teto => teto permanece, `auto_adjusted: false`
- [ ] `POST /budgets/global/auto-adjust` e idempotente para a mesma competencia
- [ ] `POST /budgets/global/auto-adjust` retorna `503` quando hf-transaction-service esta indisponivel
- [ ] `GET /budgets/global/preview-next` retorna calculo correto sem persistir

**Testes**
- [ ] Testes unitarios cobrem todos os erros da matriz com mocks de `GlobalBudgetRepository` e `HTTPClient`
- [ ] Teste unitario valida ORC-03: gasto < teto (mock httpclient retorna valor abaixo)
- [ ] Teste unitario valida ORC-04: gasto >= teto (mock httpclient retorna valor igual ou acima)
- [ ] Teste unitario valida `503` quando httpclient retorna erro
- [ ] Testes de integracao cobrem CRUD no PostgreSQL real via `testcontainers-go`
- [ ] `BusinessErrorsTotal` incrementado em cada cenario de erro nos testes

**Observabilidade**
- [ ] Log ERROR com `trace_id`, `upstream`, `status_code` quando hf-transaction-service falha
- [ ] Span OTEL criado em cada use case
- [ ] Nenhum dado sensivel logado

---

### 10. Riscos e mitigacao

**hf-transaction-service indisponivel no dia do auto-adjust**
- **Probabilidade:** media (infraestrutura pessoal, sem SLA garantido)
- **Impacto:** teto do mes nao ajustado automaticamente; usuario opera sem teto correto
- **Mitigacao:**
  - Retorna `503` + log ERROR com contexto completo
  - Metrica `upstream` no Grafana para deteccao imediata
  - Job pode ser re-executado manualmente via `POST /budgets/global/auto-adjust`
- **Plano de contingencia:** mensageria + alerta Slack/Telegram em iteracao futura

**Auto-adjust duplicado (job roda duas vezes)**
- **Probabilidade:** baixa
- **Impacto:** teto sobrescrito com o mesmo valor — comportamento correto pela idempotencia
- **Mitigacao:** logica de upsert no use case

**Primeiro mes de uso sem teto anterior**
- **Probabilidade:** alta (todo novo usuario passa por isso)
- **Impacto:** `POST /budgets/global/auto-adjust` retorna `400` com `ErrNoPreviousBudget`
- **Mitigacao:** frontend orienta o usuario a criar o teto manualmente no primeiro mes; log WARN documenta o caso
- **Plano de contingencia:** criar teto padrao baseado na receita total do mes (ORC-02) em versao futura
