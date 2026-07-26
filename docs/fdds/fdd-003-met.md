# FDD: Metas de Reducao (MET)

Versão: 1.0
Data: 2026-06-10
Domínio: MET

---

### 1. Contexto e motivação técnica

O domínio `reduction_goal` permite ao usuário definir um limite de gasto por categoria/competência e acompanhar o progresso ao longo do mês. É independente de REC e ORC em persistência, mas depende do `hf-transaction-service` em três momentos: captura do snapshot de gasto anterior (criação), gasto atual por categoria (comparativo) e gasto final (fechamento do mês).

Encaixe na arquitetura: CRUD local `Handler > UseCase > Repository > PostgreSQL` com chamadas ao hf-transaction-service via `pkg/httpclient` nos fluxos de criacao, comparativo e fechamento.

`category_id` e referencia externa sem FK local — validacao de existencia via hf-transaction-service esta marcada como futura. `user_id` tambem e referencia externa; validacao sera responsabilidade do futuro middleware JWT.

---

### 2. Objetivos tecnicos

- CRUD de metas: criar, listar, editar `target_amount`, excluir (soft delete)
- Snapshot de `previous_amount` capturado na criacao via hf-transaction-service, com fallback `null` sem bloquear a criacao
- Endpoint de comparativo `GET /goals/reduction/comparison` com degradacao parcial por categoria (MET-05/07)
- Fechamento do mes: job seta `achieved: true/false` com base em `current_month_amount <= target_amount` (MET-06)
- Chamadas ao hf-transaction-service no comparativo executadas em paralelo para minimizar latencia
- Latencia p95 < 1s no comparativo (RNF-PER-03)

**Invariantes que nao podem ser violados:**
- `target_amount` > 0 (MET-04)
- Uma meta por `(user_id, category_id, competence)` — constraint unica no banco
- `achieved` e `null` durante o mes em curso; setado apenas pelo job de fechamento

---

### 3. Escopo e exclusoes

**Incluido**
- `POST /goals/reduction` — criar meta com snapshot de `previous_amount`
- `GET /goals/reduction?user_id=&competence=` — listar metas do mes
- `PUT /goals/reduction/:id` — editar `target_amount`
- `DELETE /goals/reduction/:id` — soft delete
- `GET /goals/reduction/comparison?user_id=&competence=` — comparativo completo (MET-05/07)
- `POST /goals/reduction/close-month` — job de fechamento que seta `achieved` (MET-06)
- Preenchimento retroativo de `previous_amount: null` no comparativo

**Excluido**
- Validacao de existencia do `category_id` via hf-transaction-service (futura)
- Historico de metas por categoria ao longo de multiplos meses (analise temporal futura)
- Notificacao quando meta esta proxima de ser estourada (futura)
- Validacao de existencia do `user_id` (futuro auth service)

---

### 4. Fluxos detalhados

**Fluxo principal: `POST /goals/reduction`**
1. Handler valida campos obrigatorios: `user_id`, `category_id`, `competence`, `target_amount`
2. UseCase verifica se ja existe meta para `(user_id, category_id, competence)` — se sim, retorna `409`
3. Tenta buscar `previous_amount` via `httpclient.GetExpensesByCategory(user_id, category_id, competence_anterior)`
4. Se httpclient falhar: `previous_amount = null`, log WARN com `trace_id`, `user_id`, `category_id`, `competence`
5. Cria entidade `ReductionGoal` com `achieved: null`
6. Persiste e retorna `201`

**Fluxo principal: `GET /goals/reduction/comparison`**
1. Handler valida `user_id` e `competence`
2. UseCase busca todas as metas do `user_id` para a `competence`
3. Para cada meta, dispara chamada paralela: `httpclient.GetExpensesByCategory(user_id, category_id, competence)` → `current_month_amount`. `category_name` permanece `null` — o endpoint consumido retorna apenas `category_id` e `total`; o frontend resolve o nome pelo id (decisão HF-69)
4. Se uma categoria falha: `current_month_amount: null` naquele item, log WARN com `category_id`
5. Para metas com `previous_amount == null`: tenta preencher via httpclient (competencia anterior); se obtiver, persiste o valor
6. Calcula por categoria: `variation_pct`, `variation_label`, `on_track`, `target_progress_pct`
7. Retorna lista agregada com todos os itens (incluindo os com `current_month_amount: null`)

**Fluxo principal: `POST /goals/reduction/close-month`**
1. Handler recebe `{ "competence": "2026-06" }`
2. UseCase busca todas as metas da competencia com `achieved: null`
3. Para cada meta, busca `current_month_amount` final via httpclient
4. Se httpclient falhar para uma meta: pula, log ERROR com `goal_id` e `category_id`, continua as demais
5. Seta `achieved: true` se `current_month_amount <= target_amount`, senao `false`
6. Persiste e retorna contagem de metas fechadas

**Fluxos alternativos**
- `GET /goals/reduction` sem metas cadastradas: retorna lista vazia `[]`
- `POST /goals/reduction/close-month` sem metas com `achieved: null`: retorna `{ "closed": 0 }`
- Comparativo sem metas cadastradas: retorna lista vazia `[]`

---

### 5. Contratos publicos

**`POST /goals/reduction`**
- Rota: `POST /api/v1/goals/reduction`
- Status codes: `201` criado, `400` validacao, `409` ja existe, `500` erro interno

Request:
```json
{
  "user_id": "uuid",
  "category_id": "uuid",
  "competence": "2026-06",
  "target_amount": "400.00"
}
```

Response 201:
```json
{
  "data": {
    "id": "uuid",
    "user_id": "uuid",
    "category_id": "uuid",
    "competence": "2026-06",
    "target_amount": "400.00",
    "previous_amount": "520.00",
    "achieved": null,
    "created_at": "2026-06-10T10:00:00Z"
  },
  "error": null
}
```

---

**`GET /goals/reduction?user_id=&competence=`**
- Rota: `GET /api/v1/goals/reduction`
- Status codes: `200` ok, `400` parametros invalidos, `500` erro interno

Response 200:
```json
{
  "data": [
    {
      "id": "uuid",
      "user_id": "uuid",
      "category_id": "uuid",
      "competence": "2026-06",
      "target_amount": "400.00",
      "previous_amount": "520.00",
      "achieved": null,
      "created_at": "2026-06-10T10:00:00Z"
    }
  ],
  "error": null
}
```

---

**`PUT /goals/reduction/:id`**
- Rota: `PUT /api/v1/goals/reduction/:id`
- Status codes: `200` ok, `400` validacao, `404` nao encontrado, `500` erro interno
- Campo editavel: apenas `target_amount`

Request:
```json
{ "target_amount": "350.00" }
```

---

**`DELETE /goals/reduction/:id?requester_id=`**
- Rota: `DELETE /api/v1/goals/reduction/:id`
- Status codes: `204` deletado, `404` nao encontrado, `500` erro interno

---

**`GET /goals/reduction/comparison?user_id=&competence=`**
- Rota: `GET /api/v1/goals/reduction/comparison`
- Status codes: `200` ok (mesmo com degradacao parcial), `400` parametros invalidos, `500` erro interno

Response 200:
```json
{
  "data": [
    {
      "category_id": "uuid",
      "category_name": null,
      "previous_month_amount": "800.00",
      "current_month_amount": "620.00",
      "target_amount": "700.00",
      "on_track": true,
      "variation_pct": -22.5,
      "variation_label": "22,5% menor que o mes passado",
      "target_progress_pct": 88.6
    },
    {
      "category_id": "uuid",
      "category_name": null,
      "previous_month_amount": "300.00",
      "current_month_amount": null,
      "target_amount": "250.00",
      "on_track": null,
      "variation_pct": null,
      "variation_label": null,
      "target_progress_pct": null
    }
  ],
  "error": null
}
```

---

**`POST /goals/reduction/close-month`**
- Rota: `POST /api/v1/goals/reduction/close-month`
- Status codes: `200` ok, `400` competence invalida, `500` erro interno

Request:
```json
{ "competence": "2026-06" }
```

Response 200:
```json
{ "data": { "closed": 4 }, "error": null }
```

---

### 6. Erros, excecoes e fallback

| Condicao | Variavel Go | Rule ID | HTTP | Code |
|----------|------------|---------|------|------|
| Campo obrigatorio ausente | `ErrMissingRequiredField` | MET-04 | 400 | `MISSING_REQUIRED_FIELD` |
| `target_amount` <= 0 | `ErrInvalidTargetAmount` | MET-04 | 400 | `INVALID_TARGET_AMOUNT` |
| `competence` fora do formato `YYYY-MM` | `ErrInvalidCompetence` | MET-04 | 400 | `INVALID_COMPETENCE` |
| Ja existe meta para `(user_id, category_id, competence)` | `ErrGoalAlreadyExists` | MET-04 | 409 | `GOAL_ALREADY_EXISTS` |
| Meta nao encontrada | `ErrGoalNotFound` | — | 404 | `GOAL_NOT_FOUND` |

**Comportamentos de degradacao (nao erros HTTP):**
- hf-transaction-service falha na **criacao** → `previous_amount: null`, log WARN, meta criada com `201`
- hf-transaction-service falha no **comparativo** para uma categoria → campos calculados `null` naquele item, log WARN, demais categorias retornam normalmente
- hf-transaction-service falha no **fechamento** para uma meta → pula aquela meta, log ERROR com `goal_id` e `category_id`, continua as demais

---

### 7. Observabilidade

**Spans OTEL** (criados no use case)
- `reduction_goal.create`
- `reduction_goal.list`
- `reduction_goal.update`
- `reduction_goal.delete`
- `reduction_goal.comparison`
- `reduction_goal.close_month`

**Metricas**
```go
// MET-04 — validacao
metrics.BusinessErrorsTotal.WithLabelValues("MET", "MET-04").Inc()
// upstream — falha ao buscar gasto por categoria
metrics.BusinessErrorsTotal.WithLabelValues("MET", "upstream").Inc()
```

**Campos de log obrigatorios**
- Entrada (INFO): `operation`, `user_id`, `competence` (`close_month` e um job por competencia, sem `user_id` na entrada)
- Saida (INFO): `operation`, `duration_ms`, `goal_id` (quando aplicavel), `categories_count` (no comparativo)
- Degradacao na criacao (WARN): `trace_id`, `user_id`, `category_id`, `competence`, `reason`
- Degradacao no comparativo (WARN): `trace_id`, `user_id`, `category_id`, `competence`
- Falha no fechamento (ERROR): `trace_id`, `goal_id`, `category_id`, `competence`

**Alertas minimos no Grafana**
- `business_errors_total{domain="MET", rule_id="upstream"}` > 5 em 10 minutos

---

### 8. Dependencias e compatibilidade

| Componente | Versao minima | Observacoes |
|-----------|--------------|-------------|
| Go | 1.23 | |
| PostgreSQL | 16 | |
| GORM | v2 | |
| shopspring/decimal | v1 | Obrigatorio para `target_amount` e `previous_amount` |
| hf-transaction-service | — | Contrato em INTEGRATIONS.md |

**Dependencias entre use cases**
- MET nao depende de REC nem ORC em persistencia
- `reduction_goal/comparison.go` usa `httpclient.GetExpensesByCategory` — chamadas paralelas com `sync.WaitGroup` ou `errgroup`

**Evolucao futura do contrato com hf-transaction-service**
- Endpoint batch para buscar multiplas categorias em uma unica chamada reduziria latencia do comparativo

**Indices recomendados**
```sql
CREATE INDEX idx_reduction_goals_user_competence ON reduction_goals (user_id, competence) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_reduction_goals_user_category_competence ON reduction_goals (user_id, category_id, competence) WHERE deleted_at IS NULL;
```

---

### 9. Criterios de aceite tecnicos

**Funcional**
- [ ] `POST /goals/reduction` cria meta com `previous_amount` preenchido quando hf-transaction-service disponivel
- [ ] `POST /goals/reduction` cria meta com `previous_amount: null` quando hf-transaction-service indisponivel, sem retornar erro
- [ ] `POST /goals/reduction` retorna `409` se ja existe meta para `(user_id, category_id, competence)`
- [ ] `GET /goals/reduction/comparison` retorna `current_month_amount: null` por categoria quando upstream falha, sem falhar o request
- [ ] `GET /goals/reduction/comparison` preenche `previous_amount: null` retroativamente quando upstream disponivel
- [ ] `PUT /goals/reduction/:id` atualiza apenas `target_amount`
- [ ] `POST /goals/reduction/close-month` seta `achieved: true` quando `current_month_amount <= target_amount` (MET-06)
- [ ] `POST /goals/reduction/close-month` pula metas com falha upstream e continua as demais

**Testes**
- [ ] Testes unitarios cobrem todos os erros da matriz com mocks de `ReductionGoalRepository` e `HTTPClient`
- [ ] Teste unitario valida criacao com `previous_amount: null` quando mock httpclient retorna erro
- [ ] Teste unitario valida degradacao parcial no comparativo (uma categoria falha, demais retornam)
- [ ] Teste unitario valida MET-06: `achieved: true` (gasto <= meta) e `achieved: false` (gasto > meta)
- [ ] Testes de integracao cobrem CRUD e soft delete no PostgreSQL real via `testcontainers-go`
- [ ] `BusinessErrorsTotal` incrementado em cada cenario de erro nos testes

**Observabilidade**
- [ ] Log WARN com `category_id` quando `previous_amount` nao pôde ser capturado na criacao
- [ ] Log ERROR com `goal_id` quando fechamento falha para uma meta especifica
- [ ] Span OTEL criado em cada use case
- [ ] Nenhum dado sensivel logado

---

### 10. Riscos e mitigacao

**`previous_amount` nunca preenchido**
- **Probabilidade:** baixa
- **Impacto:** comparativo exibe `variation_pct` e `variation_label` nulos — experiencia degradada
- **Mitigacao:**
  - Tentativa de preenchimento retroativo no comparativo
  - Log WARN persistente para rastrear metas sem snapshot
- **Plano de contingencia:** endpoint administrativo futuro para reprocessar snapshots em lote

**Fechamento do mes com hf-transaction-service indisponivel**
- **Probabilidade:** media (job roda no fim do mes)
- **Impacto:** `achieved` permanece `null` — mes nao fechado corretamente
- **Mitigacao:**
  - Job pula metas com falha e loga ERROR por `goal_id`
  - Job pode ser re-executado manualmente
- **Plano de contingencia:** alerta Slack/Telegram em iteracao futura

**Latencia acumulada no comparativo (multiplas chamadas upstream)**
- **Probabilidade:** alta (uma chamada por categoria)
- **Impacto:** latencia pode ultrapassar 1s com muitas metas
- **Mitigacao:** chamadas paralelas com `errgroup`; timeout de 5s por chamada
- **Plano de contingencia:** endpoint batch no hf-transaction-service para buscar multiplas categorias em uma unica chamada
