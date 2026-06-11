# FDD: Saldo Mensal (SAL)

Versão: 1.0
Data: 2026-06-10
Domínio: SAL

---

### 1. Contexto e motivação técnica

O domínio `balance` fornece o saldo mensal calculado sob demanda — sem persistencia. Agrega dados locais deste servico (receitas e teto global) com dados do `hf-transaction-service` (despesas realizadas e contas a pagar pendentes).

Encaixe na arquitetura: `Handler > UseCase > IncomeRepository (local) + GlobalBudgetRepository (local) + HTTPClient (hf-transaction-service)`. Nao ha tabela de `balance` no banco.

Ordem de implementacao obrigatoria: SAL depende de REC (`IncomeRepository`) e ORC (`GlobalBudgetRepository`) estarem implementados.

Politica de falha: diferente do MET, qualquer falha no hf-transaction-service retorna erro imediato ao cliente — SAL nao retorna dados parciais.

`user_id` e referencia externa. Validacao sera responsabilidade do futuro middleware JWT.

---

### 2. Objetivos tecnicos

- Endpoint unico `GET /balance?user_id=&competence=` que agrega 4 fontes em paralelo
- Chamadas HTTP ao hf-transaction-service executadas com `errgroup` para minimizar latencia
- Formulas (SAL-01 a SAL-05, ORC-06/07):
  - `balance_today = total_income - total_expenses`
  - `committed_bills = sum(pending accounts)`
  - `projected_balance = balance_today - committed_bills`
  - `is_projected_negative = projected_balance < 0`
  - `ceiling_usage_pct = int(total_expenses / ceiling * 100)` (null se sem teto)
  - `ceiling_exceeded = total_expenses > ceiling` (false se sem teto)
- `total_income = 0` retorna response normalmente com `balance_today` negativo ou zero
- Latencia p95 < 1s (RNF-PER-03)

**Invariantes que nao podem ser violados:**
- `balance_today`, `projected_balance` e `is_projected_negative` sempre presentes no response, mesmo que zero (SAL-04/05)
- Nunca retornar dados parciais sem indicar qual upstream falhou

---

### 3. Escopo e exclusoes

**Incluido**
- `GET /balance?user_id=&competence=` — unico endpoint do dominio
- Agregacao de `total_income` (local), `total_expenses`, `total_personal`, `total_shared` e `committed_bills` (upstream)
- Campos ORC: `ceiling`, `ceiling_usage_pct`, `ceiling_exceeded` (quando teto existe)
- Flag `is_projected_negative` para alerta visual no frontend (SAL-05)
- Chamadas upstream em paralelo com `errgroup`

**Excluido**
- Persistencia de qualquer dado de saldo
- Historico de saldos por competencia (futura analise temporal)
- Breakdown por categoria de despesa (usa MET para isso)
- Cache do response (futura otimizacao)
- Validacao de existencia do `user_id` (futuro auth service)

---

### 4. Fluxos detalhados

**Fluxo unico: `GET /balance?user_id=&competence=`**
1. Handler valida `user_id` e `competence` (formato `YYYY-MM`)
2. UseCase dispara em paralelo via `errgroup`:
   - `IncomeRepository.SumByUserAndCompetence(ctx, user_id, competence)` → `total_income`
   - `GlobalBudgetRepository.GetByUserAndCompetence(ctx, user_id, competence)` → `ceiling` (nil se nao existe)
   - `httpclient.GetExpenseTotals(user_id, competence)` → `total_personal`, `total_shared`, `total_general`
   - `httpclient.GetAccountsPayable(user_id, lastDayOfMonth(competence))` → lista de contas pendentes
3. Se qualquer chamada HTTP falhar → `errgroup` cancela contexto e retorna erro imediatamente (503 ou 502)
4. Calcula `committed_bills = sum(account.amount for all pending accounts)`
5. Calcula campos derivados:
   - `balance_today = total_income - total_general`
   - `projected_balance = balance_today - committed_bills`
   - `is_projected_negative = projected_balance < 0`
   - Se `ceiling != nil`: `ceiling_usage_pct = int(total_general / ceiling * 100)`, `ceiling_exceeded = total_general > ceiling`
   - Se `ceiling == nil`: `ceiling_usage_pct = null`, `ceiling_exceeded = false`
6. Retorna `200` com response completo

**Diagrama de sequencia:**
```
Client > Handler > UseCase ──┬──> IncomeRepository ──────────────> PostgreSQL
                             ├──> GlobalBudgetRepository ──────────> PostgreSQL
                             ├──> HTTPClient ──> GET /expenses/user/:id/totals ──> hf-transaction-service
                             └──> HTTPClient ──> GET /accounts-payable ──────────> hf-transaction-service
```

**Calculo de `lastDayOfMonth`:**
```go
func lastDayOfMonth(competence string) (string, error) {
    t, err := time.Parse("2006-01", competence)
    if err != nil {
        return "", err
    }
    lastDay := time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, time.UTC)
    return lastDay.Format("2006-01-02"), nil
}
```

**Fluxos alternativos**
- `total_income = 0` (sem receitas no mes): retorna `200` normalmente com `balance_today` igual ao negativo das despesas
- `ceiling = nil` (sem teto cadastrado): `ceiling`, `ceiling_usage_pct` retornam `null`; `ceiling_exceeded = false`
- Falha em `GetExpenseTotals`: retorna `503` com `UPSTREAM_TIMEOUT` ou `502` com `UPSTREAM_ERROR`
- Falha em `GetAccountsPayable`: mesma politica de erro

---

### 5. Contratos publicos

**`GET /balance?user_id=&competence=`**
- Rota: `GET /api/v1/balance`
- Status codes: `200` ok, `400` validacao, `502` resposta inesperada do upstream, `503` upstream indisponivel/timeout, `500` erro interno

Response 200 (com teto cadastrado):
```json
{
  "data": {
    "user_id": "uuid",
    "competence": "2026-06",
    "total_income": "7500.00",
    "total_personal": "1800.00",
    "total_shared": "1400.00",
    "total_expenses": "3200.00",
    "balance_today": "4300.00",
    "committed_bills": "850.00",
    "projected_balance": "3450.00",
    "is_projected_negative": false,
    "ceiling": "5000.00",
    "ceiling_usage_pct": 64,
    "ceiling_exceeded": false
  },
  "error": null
}
```

Response 200 (sem teto cadastrado, sem receitas):
```json
{
  "data": {
    "user_id": "uuid",
    "competence": "2026-06",
    "total_income": "0.00",
    "total_personal": "0.00",
    "total_shared": "0.00",
    "total_expenses": "0.00",
    "balance_today": "0.00",
    "committed_bills": "0.00",
    "projected_balance": "0.00",
    "is_projected_negative": false,
    "ceiling": null,
    "ceiling_usage_pct": null,
    "ceiling_exceeded": false
  },
  "error": null
}
```

Response 503:
```json
{
  "data": null,
  "error": { "code": "UPSTREAM_TIMEOUT", "message": "expense service unavailable, please retry" }
}
```

Response 502:
```json
{
  "data": null,
  "error": { "code": "UPSTREAM_ERROR", "message": "unexpected response from expense service" }
}
```

---

### 6. Erros, excecoes e fallback

| Condicao | Variavel Go | Rule ID | HTTP | Code |
|----------|------------|---------|------|------|
| `user_id` ausente ou invalido | `ErrMissingRequiredField` | SAL-01 | 400 | `MISSING_REQUIRED_FIELD` |
| `competence` fora do formato `YYYY-MM` | `ErrInvalidCompetence` | SAL-01 | 400 | `INVALID_COMPETENCE` |
| hf-transaction-service timeout (> 5s) | `ErrUpstreamTimeout` | — | 503 | `UPSTREAM_TIMEOUT` |
| hf-transaction-service retornou status inesperado | `ErrUpstreamError` | — | 502 | `UPSTREAM_ERROR` |

**Resiliencia:**
- Timeout: 5s por chamada HTTP
- As duas chamadas HTTP rodam em paralelo via `errgroup` — se qualquer uma falhar, contexto e cancelado e erro retornado imediatamente
- Sem fallback — SAL nunca retorna dados parciais
- Falhas locais (IncomeRepository, GlobalBudgetRepository) retornam `500`

---

### 7. Observabilidade

**Spans OTEL** (criados no use case)
- `balance.get` — span pai que engloba toda a operacao
- `balance.get.expense_totals` — sub-span da chamada a `GET /expenses/user/:id/totals`
- `balance.get.accounts_payable` — sub-span da chamada a `GET /accounts-payable`

**Metricas**
```go
// SAL-01 — validacao de entrada
metrics.BusinessErrorsTotal.WithLabelValues("SAL", "SAL-01").Inc()
// upstream — falha nas chamadas HTTP
metrics.BusinessErrorsTotal.WithLabelValues("SAL", "upstream").Inc()
```

**Campos de log obrigatorios**
- Entrada (INFO): `operation`, `user_id`, `competence`
- Saida (INFO): `operation`, `duration_ms`, `balance_today`, `projected_balance`, `is_projected_negative`
- Erro upstream (ERROR): `trace_id`, `upstream_endpoint`, `status_code`, `user_id`, `competence`

**Alertas minimos no Grafana**
- `business_errors_total{domain="SAL", rule_id="upstream"}` > 3 em 5 minutos

---

### 8. Dependencias e compatibilidade

| Componente | Versao minima | Observacoes |
|-----------|--------------|-------------|
| Go | 1.23 | |
| golang.org/x/sync/errgroup | — | Para chamadas paralelas com cancelamento |
| shopspring/decimal | v1 | Todos os campos monetarios |
| hf-transaction-service | — | Contrato em INTEGRATIONS.md |

**Dependencias entre use cases (ordem obrigatoria de implementacao)**
- `IncomeRepository.SumByUserAndCompetence` — implementado em REC
- `GlobalBudgetRepository.GetByUserAndCompetence` — implementado em ORC
- SAL so pode ser implementado apos REC e ORC estarem prontos

**Impacto de mudancas**
- Adicao de campo no response: nao-breaking
- Remocao ou renomeacao de campo: breaking change — coordenar com frontend
- Mudanca no contrato do hf-transaction-service: atualizar INTEGRATIONS.md antes de implementar

---

### 9. Criterios de aceite tecnicos

**Funcional**
- [ ] `GET /balance` retorna todos os campos obrigatorios mesmo com `total_income = 0` (SAL-04)
- [ ] `balance_today = total_income - total_expenses` calculado corretamente (SAL-01)
- [ ] `projected_balance = balance_today - committed_bills` calculado corretamente (SAL-03)
- [ ] `is_projected_negative: true` quando `projected_balance < 0` (SAL-05)
- [ ] `ceiling_usage_pct` retornado como inteiro (ex: `64`) quando teto existe
- [ ] `ceiling` e `ceiling_usage_pct` retornam `null` quando sem teto cadastrado
- [ ] `total_personal` e `total_shared` presentes no response separadamente
- [ ] Falha em qualquer chamada HTTP retorna `503` ou `502` — nunca dados parciais
- [ ] `lastDayOfMonth("2026-06")` retorna `"2026-06-30"`
- [ ] `lastDayOfMonth("2026-02")` retorna `"2026-02-28"` (e `"2024-02"` retorna `"2024-02-29"`)

**Testes**
- [ ] Teste unitario: cenario completo com teto — todos os campos calculados corretamente
- [ ] Teste unitario: cenario sem teto — `ceiling = null`, `ceiling_usage_pct = null`
- [ ] Teste unitario: `total_income = 0` — retorna `200` normalmente
- [ ] Teste unitario: falha em `GetExpenseTotals` → `503`
- [ ] Teste unitario: falha em `GetAccountsPayable` → `503`
- [ ] Teste unitario: resposta inesperada do upstream → `502`
- [ ] Teste unitario: `lastDayOfMonth` cobre fevereiro, ano bissexto e dezembro
- [ ] `BusinessErrorsTotal` incrementado nos cenarios de erro

**Observabilidade**
- [ ] Span pai `balance.get` com sub-spans `balance.get.expense_totals` e `balance.get.accounts_payable`
- [ ] Log INFO na saida com `balance_today`, `projected_balance`, `is_projected_negative`
- [ ] Log ERROR com `upstream_endpoint` e `status_code` em falha upstream

---

### 10. Riscos e mitigacao

**hf-transaction-service indisponivel**
- **Probabilidade:** media
- **Impacto:** alto — `/balance` e o endpoint central do frontend; indisponibilidade bloqueia o painel principal
- **Mitigacao:**
  - Timeout de 5s + retorno imediato de `503`
  - Alerta Grafana `business_errors_total{domain="SAL", rule_id="upstream"}` > 3 em 5 minutos
- **Plano de contingencia:** cache de curta duracao (TTL 60s) em iteracao futura para servir ultimo saldo conhecido

**Divergencia entre `lastDayOfMonth` e filtro real do hf-transaction-service**
- **Probabilidade:** baixa
- **Impacto:** `committed_bills` incorreto — saldo projetado errado
- **Mitigacao:**
  - Teste unitario especifico cobrindo fevereiro, anos bissextos e dezembro
  - Contrato documentado e versionado em INTEGRATIONS.md
- **Plano de contingencia:** revisitar INTEGRATIONS.md se hf-transaction-service alterar o endpoint

**Latencia acumulada das chamadas paralelas**
- **Probabilidade:** baixa (apenas 2 chamadas HTTP simultaneas)
- **Impacto:** ultrapassar p95 de 1s em caso de degradacao do hf-transaction-service
- **Mitigacao:** `errgroup` com contexto cancelavel; timeout individual de 5s; histograma de duracao no Grafana
- **Plano de contingencia:** cache do response com TTL curto em iteracao futura
