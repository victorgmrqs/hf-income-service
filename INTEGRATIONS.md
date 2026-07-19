# Integrações — hf-income-service

## Dependências externas

Este serviço consome a API do `hf-transaction-service` para o auto-ajuste do teto global (domínio ORC) e para calcular saldo e despesas comprometidas (domínio SAL).

Toda comunicação é HTTP REST. O cliente HTTP está em `src/pkg/httpclient/client.go` e é injetado via interface — nunca chamado diretamente nos use cases.

---

## Interface do cliente HTTP

```go
// src/pkg/httpclient/transaction_client.go
type TransactionClient interface {
    GetExpenseTotals(ctx context.Context, userID, competence string) (*ExpenseTotalsOutput, error)
    GetAccountsPayable(ctx context.Context, userID, dueDateUntil string) ([]PendingBillOutput, error)
    GetExpensesByCategory(ctx context.Context, userID, categoryID, competence string) (*ExpensesByCategoryOutput, error)
}
```

Os use cases dependem apenas dessa interface — o cliente HTTP concreto (`NewTransactionClient`) é injetado em `cmd/server/main.go` (`TRANSACTION_SERVICE_URL`). Timeout de **5s por chamada** via `context.WithTimeout`; timeout → `ErrUpstreamTimeout`, status inesperado → `ErrUpstreamError` (HF-57/HF-41).

> Consolidação (HF-41): o `client.go` genérico do HF-44 foi removido — `transaction_client.go` é o cliente canônico único.

---

## Endpoint 1 — Totais de despesas

**Usado em:**
- ORC-03/04 (`auto-adjust` e `preview-next` — gasto do mês anterior/corrente) — HF-44
- SAL-01 (`total_expenses`, `total_personal`, `total_shared` no response de `/balance`) — HF-41

**Contrato:**

```
GET /api/v1/expenses/user/{user_id}/totals?competence=YYYY-MM
```

**Response (200):**

```json
{
  "data": {
    "competence": "2026-06",
    "total_personal": 1800.00,
    "total_shared": 1400.00,
    "total_general": 3200.00
  },
  "error": null
}
```

| Campo | Descrição |
|-------|-----------|
| `total_personal` | Despesas exclusivas do usuário na competência |
| `total_shared` | Parte do usuário nas despesas compartilhadas (`divided_amount`) |
| `total_general` | `total_personal + total_shared` |

**Mapeamento para SAL-01:**

```
balance.total_expenses = total_general
```

---

## Endpoint 2 — Contas a pagar pendentes

**Usado em:** SAL-02 e SAL-03 (`committed_bills` e `projected_balance` no response de `/balance`) — HF-41 (`GetAccountsPayable` + `LastDayOfMonth`)

**Contrato:**

```
GET /api/v1/accounts-payable?user_id={uuid}&status=PENDING&due_date_until={YYYY-MM-DD}
```

> **Atenção:** Este endpoint não aceita `competence` como parâmetro. Para filtrar contas da competência atual, use `due_date_until` com o **último dia do mês** da competência.

**Cálculo do `due_date_until` a partir da competência:**

```go
// competence = "2026-06"
// due_date_until = "2026-06-30" (último dia do mês)
func lastDayOfMonth(competence string) (string, error) {
    t, err := time.Parse("2006-01", competence)
    if err != nil {
        return "", err
    }
    lastDay := time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, time.UTC)
    return lastDay.Format("2006-01-02"), nil
}
```

**Response (200):**

```json
{
  "data": [
    {
      "id": "uuid",
      "description": "Escola",
      "amount": 850.00,
      "due_date": "2026-06-10T00:00:00Z",
      "status": "PENDING",
      "recurrence": "MONTHLY",
      "paid_at": null,
      "expense_id": null
    }
  ],
  "error": null
}
```

**Mapeamento para SAL-02/SAL-03:**

```
balance.committed_bills  = sum(account.amount for account in pending_bills)
balance.projected_balance = balance.balance_today - balance.committed_bills
```

---

## Endpoint 3 — Despesas por categoria (placeholder MET)

**Usado em:** MET (domínio de metas de redução — não implementado ainda)

**Contrato (provisório):**

```
GET /api/v1/expenses/user/{user_id}/by-category?category_id={uuid}&competence=YYYY-MM
```

**Response (200):**

```json
{
  "data": { "category_id": "uuid", "total": 500.00 },
  "error": null
}
```

> Contrato sujeito a revisão quando fdd-003-met.md definir o uso completo. (HF-57)

---

## Variáveis de ambiente

```env
TRANSACTION_SERVICE_URL=http://hf-transaction-service:8080
```

---

## Comportamento em falha

| Cenário | Comportamento esperado |
|---------|----------------------|
| `hf-transaction-service` indisponível | Retornar erro `503 Service Unavailable` com code `UPSTREAM_UNAVAILABLE` |
| Timeout na chamada | Timeout de 5s; retornar `503` com code `UPSTREAM_TIMEOUT` |
| Response inesperado (não-200) | Logar o status code recebido; retornar `502 Bad Gateway` |

O use case `balance/get.go` não deve silenciar erros do cliente HTTP — falha no upstream = falha na request.
