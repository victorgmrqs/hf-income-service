# FDD: Receitas (REC)

Versão: 1.0
Data: 2026-06-09
Domínio: REC

---

### 1. Contexto e motivação técnica

O domínio `income` gerencia as receitas mensais de cada usuário (salário, freelance, investimento, aluguel, outros). É o domínio mais independente do serviço — sem chamadas externas — e bloqueia os domínios SAL (saldo) e ORC (teto global), que dependem do total de receitas para seus cálculos.

Encaixe na arquitetura: fluxo padrão `Handler > UseCase > Repository > PostgreSQL`, sem HTTPClient.

`user_id` é uma referência externa sem FK local. A validação de existência do usuário será responsabilidade do futuro middleware JWT (serviço de auth). Controle de acesso por papel/persona está fora do escopo atual.

---

### 2. Objetivos técnicos

- CRUD completo de receitas com soft delete
- Listagem por `user_id + competence` com `total_income` agregado no response
- Propagação idempotente de receitas recorrentes via `POST /income/propagate`
- Preservação de histórico: edição de receita recorrente nao altera competencias anteriores
- Latencia p95 < 300ms nos endpoints de leitura (RNF-PER-01)

**Invariantes que nao podem ser violados:**
- `amount` nunca pode ser zero ou negativo em persistencia (REC-02)
- Registro com `origin_id` preenchido e imutavel — e uma copia propagada (REC-03)

---

### 3. Escopo e exclusoes

**Incluido**
- `POST /income` — criar receita
- `GET /income?user_id=&competence=` — listar com total agregado
- `GET /income/:id` — buscar por ID
- `PUT /income/:id` — editar (campos permitidos; registros propagados sao imutaveis)
- `DELETE /income/:id` — soft delete
- `POST /income/propagate` — propagar recorrentes para proxima competencia (idempotente)

**Excluido**
- Receita compartilhada / fundo familiar (REC-05 — funcionalidade futura)
- Validacao de existencia do `user_id` (responsabilidade do futuro auth service)
- Controle de acesso por papel/persona (futuro middleware JWT)
- Edicao retroativa de competencias passadas

---

### 4. Fluxos detalhados

**Fluxo principal: `POST /income`**
1. Handler valida campos obrigatorios do body: `description`, `amount`, `date`, `competence`, `type`, `user_id`
2. Chama `IncomeUseCase.Create(ctx, input)`
3. UseCase valida: `amount > 0` (REC-02), `type` pertence aos valores validos (REC-01), `competence` no formato `YYYY-MM` (REC-01)
4. UseCase cria entidade `Income`; UUID gerado no `BeforeCreate`
5. Chama `IncomeRepository.Create(ctx, income)`
6. Retorna `201` com o registro criado

**Fluxo principal: `PUT /income/:id`**
1. Handler extrai `id` da rota e body com campos editaveis
2. UseCase busca o registro — se nao existir ou estiver soft-deleted, retorna `404`
3. Se `origin_id != nil`, bloqueia a edicao com `ErrCannotEditPropagatedIncome` (REC-03)
4. Aplica alteracoes apenas no registro da competencia atual — nao toca registros anteriores (REC-03)
5. Persiste e retorna `200`

**Campos editaveis em `PUT /income/:id`:** `description`, `amount`, `date`, `type`, `recurrent`
**Campos imutaveis:** `user_id`, `origin_id`, `competence`

**Fluxo principal: `POST /income/propagate`**
1. Handler recebe `{ "competence": "YYYY-MM" }` (competencia de destino)
2. UseCase busca todas as receitas com `recurrent: true` da competencia anterior (`competence - 1 mes`)
3. Para cada uma, verifica se ja existe registro com `origin_id = income.ID` e `competence = destino`
4. Se ja existe, ignora silenciosamente (idempotencia)
5. Se nao existe, cria nova entrada com `origin_id` = ID da original e `competence` = destino
6. Retorna `200` com contagem de registros propagados

**Fluxos alternativos**
- `GET /income/:id` com ID inexistente ou soft-deleted: retorna `404`
- `DELETE /income/:id` ja deletado: retorna `404`
- `POST /income/propagate` sem receitas recorrentes na competencia anterior: retorna `200` com `{ "propagated": 0 }`

---

### 5. Contratos publicos

**`POST /income`**
- Rota: `POST /api/v1/income`
- Status codes: `201` criado, `400` validacao, `500` erro interno

Request:
```json
{
  "user_id": "uuid",
  "description": "Salario junho",
  "amount": "5000.00",
  "date": "2026-06-05",
  "competence": "2026-06",
  "type": "SALARY",
  "recurrent": false
}
```

Response 201:
```json
{
  "data": {
    "id": "uuid",
    "user_id": "uuid",
    "description": "Salario junho",
    "amount": "5000.00",
    "date": "2026-06-05",
    "competence": "2026-06",
    "type": "SALARY",
    "recurrent": false,
    "origin_id": null,
    "created_at": "2026-06-09T10:00:00Z"
  },
  "error": null
}
```

---

**`GET /income?user_id=&competence=`**
- Rota: `GET /api/v1/income`
- Status codes: `200` ok, `400` parametros invalidos, `500` erro interno

Response 200:
```json
{
  "data": {
    "competence": "2026-06",
    "total_income": "7500.00",
    "items": [
      {
        "id": "uuid",
        "user_id": "uuid",
        "description": "Salario junho",
        "amount": "5000.00",
        "date": "2026-06-05",
        "competence": "2026-06",
        "type": "SALARY",
        "recurrent": false,
        "origin_id": null,
        "created_at": "2026-06-09T10:00:00Z"
      }
    ]
  },
  "error": null
}
```

---

**`GET /income/:id?requester_id=`**
- Rota: `GET /api/v1/income/:id`
- Status codes: `200` ok, `404` nao encontrado, `500` erro interno

---

**`PUT /income/:id`**
- Rota: `PUT /api/v1/income/:id`
- Status codes: `200` ok, `400` validacao ou registro propagado, `404` nao encontrado, `500` erro interno

---

**`DELETE /income/:id?requester_id=`**
- Rota: `DELETE /api/v1/income/:id`
- Status codes: `204` deletado, `404` nao encontrado, `500` erro interno

---

**`POST /income/propagate`**
- Rota: `POST /api/v1/income/propagate`
- Status codes: `200` ok (mesmo que nenhum propagado), `400` competence invalida, `500` erro interno

Request:
```json
{ "competence": "2026-07" }
```

Response 200:
```json
{ "data": { "propagated": 3 }, "error": null }
```

Response 400:
```json
{ "data": null, "error": { "code": "INVALID_COMPETENCE", "message": "competence must be in YYYY-MM format" } }
```

---

### 6. Erros, excecoes e fallback

| Condicao | Variavel Go | Rule ID | HTTP | Code |
|----------|------------|---------|------|------|
| Campo obrigatorio ausente | `ErrMissingRequiredField` | REC-01 | 400 | `MISSING_REQUIRED_FIELD` |
| `amount` <= 0 | `ErrInvalidAmount` | REC-02 | 400 | `INVALID_AMOUNT` |
| `type` fora dos valores validos | `ErrInvalidIncomeType` | REC-01 | 400 | `INVALID_INCOME_TYPE` |
| `competence` fora do formato `YYYY-MM` | `ErrInvalidCompetence` | REC-01 | 400 | `INVALID_COMPETENCE` |
| Receita nao encontrada | `ErrIncomeNotFound` | — | 404 | `INCOME_NOT_FOUND` |
| Tentativa de editar registro propagado (`origin_id != nil`) | `ErrCannotEditPropagatedIncome` | REC-03 | 400 | `CANNOT_EDIT_PROPAGATED_INCOME` |

Sem chamadas externas neste dominio — nao ha estrategia de fallback necessaria.

---

### 7. Observabilidade

**Spans OTEL** (criados no use case)
- `income.create`
- `income.get`
- `income.list`
- `income.update`
- `income.delete`
- `income.propagate`

**Metricas**
```go
// REC-01
metrics.BusinessErrorsTotal.WithLabelValues("REC", "REC-01").Inc()
// REC-02
metrics.BusinessErrorsTotal.WithLabelValues("REC", "REC-02").Inc()
// REC-03
metrics.BusinessErrorsTotal.WithLabelValues("REC", "REC-03").Inc()
```

**Campos de log obrigatorios**
- Entrada (INFO): `operation`, `user_id`, `competence`
- Saida (INFO): `operation`, `duration_ms`, `income_id` (quando aplicavel)
- Erro/warn (WARN): `trace_id`, `rule_id`, `reason`

**Alerta minimo no Grafana**
- `business_errors_total{domain="REC"}` > 10 em 5 minutos

---

### 8. Dependencias e compatibilidade

| Componente | Versao minima | Observacoes |
|-----------|--------------|-------------|
| Go | 1.23 | |
| PostgreSQL | 16 | |
| GORM | v2 | |
| shopspring/decimal | v1 | Obrigatorio para `amount` — nunca float64 |

**Dependencias entre use cases**
- `balance/get.go` usa `IncomeRepository.ListByUserAndCompetence` para calcular `total_income`
- `global_budget/auto_adjust.go` pode usar o total de receitas da competencia anterior como referencia para o teto sugerido

**Indice recomendado**
```sql
CREATE INDEX idx_incomes_user_competence ON incomes (user_id, competence, deleted_at);
CREATE UNIQUE INDEX idx_incomes_origin_competence ON incomes (origin_id, competence) WHERE origin_id IS NOT NULL;
```

---

### 9. Criterios de aceite tecnicos

**Funcional**
- [ ] `POST /income` cria receita e retorna `201` com o registro completo
- [ ] `GET /income?user_id=&competence=` retorna lista de itens + `total_income` agregado
- [ ] `GET /income/:id` retorna `404` para ID inexistente ou soft-deleted
- [ ] `PUT /income/:id` retorna `400` ao tentar editar registro com `origin_id` preenchido
- [ ] `DELETE /income/:id` aplica soft delete; registro nao aparece em listagens subsequentes
- [ ] `POST /income/propagate` e idempotente: segunda execucao para a mesma competencia nao cria duplicatas

**Testes**
- [ ] Testes unitarios cobrem todos os 6 erros da matriz com mocks de `IncomeRepository`
- [ ] Testes de integracao com PostgreSQL real via `testcontainers-go` cobrem CRUD e soft delete
- [ ] Teste de integracao valida idempotencia da propagacao
- [ ] `BusinessErrorsTotal` incrementado em cada cenario de erro nos testes

**Observabilidade**
- [ ] Span OTEL criado em cada use case
- [ ] Log INFO na entrada e saida de cada operacao
- [ ] Nenhum campo sensivel logado isoladamente sem contexto de operacao

---

### 10. Riscos e mitigacao

**Propagacao duplicada**
- **Probabilidade:** media (job pode ser re-executado por falha de infraestrutura)
- **Impacto:** dados duplicados, `total_income` inflado no SAL
- **Mitigacao:**
  - Constraint unica em `(origin_id, competence)` no banco
  - Verificacao no use case antes de inserir
- **Plano de contingencia:** script de limpeza por `origin_id + competence` duplicados

**Edicao retroativa acidental**
- **Probabilidade:** baixa (regra clara no use case)
- **Impacto:** historico corrompido, saldos de meses anteriores alterados
- **Mitigacao:**
  - `ErrCannotEditPropagatedIncome` bloqueado no use case
  - Validacao reforecada por teste de integracao
- **Plano de contingencia:** soft delete + recriar registro correto

**Crescimento da tabela `incomes` por recorrencia**
- **Probabilidade:** baixa (volume pessoal, < 1.000 registros/mes por usuario)
- **Impacto:** degradacao de query de listagem
- **Mitigacao:** indice composto em `(user_id, competence, deleted_at)`
- **Plano de contingencia:** particao da tabela por `competence` se volume crescer
