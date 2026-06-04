# Especificação da API — hf-income-service

## Informações Gerais

| Item | Valor |
|---|---|
| Base URL | `http://localhost:8081/api/v1` |
| Formato | JSON |
| Autenticação | Nenhuma no MVP (futura: JWT via hf-auth-service) |
| Versão | v1 |

## Padrão de Response

**Sucesso:**
```json
{ "data": <payload>, "error": null }
```

**Erro:**
```json
{ "data": null, "error": { "code": "VALIDATION_ERROR", "message": "descrição legível" } }
```

## Códigos HTTP

| Código | Uso |
|---|---|
| 200 | GET/PUT bem-sucedido |
| 201 | POST bem-sucedido |
| 204 | DELETE bem-sucedido |
| 400 | Erro de validação / dados inválidos |
| 404 | Recurso não encontrado |
| 409 | Conflito (ex: teto já existe para a competência) |
| 500 | Erro interno |

---

## Receitas (REC)

### `POST /income`
Criar uma receita.

**Body:**
```json
{
  "user_id": "uuid",
  "description": "Salário Empresa X",
  "amount": 7500.00,
  "type": "SALARY",
  "date": "2026-06-05",
  "competence": "2026-06",
  "recurrent": true
}
```

### `GET /income?user_id=&competence=`
Listar receitas do usuário na competência.

**Query params:**
- `user_id` (obrigatório)
- `competence` (obrigatório, formato YYYY-MM)
- `type` (opcional, filtro por tipo)

### `GET /income/:id?requester_id=`
Buscar receita por ID.

### `PUT /income/:id`
Editar receita. Aplica a partir da competência atual — não retroativo (REC-03).

**Body:**
```json
{
  "requester_id": "uuid",
  "description": "Salário Reajustado",
  "amount": 8200.00
}
```

### `DELETE /income/:id?requester_id=`
Soft delete da receita.

### `POST /income/propagate`
Propagar receitas recorrentes para a próxima competência (REC-04). Chamado por job agendado no 1º do mês.

**Body:**
```json
{ "competence": "2026-07" }
```

---

## Teto Global (ORC)

### `POST /budgets/global`
Criar teto global para uma competência.

**Body:**
```json
{
  "user_id": "uuid",
  "competence": "2026-06",
  "ceiling": 5000.00
}
```

### `GET /budgets/global?user_id=&competence=`
Buscar teto da competência.

### `PUT /budgets/global/:id`
Editar teto manualmente (seta `auto_adjusted: false`).

**Body:**
```json
{ "ceiling": 4800.00 }
```

### `GET /budgets/global/preview-next?user_id=`
Preview do auto-ajuste para o próximo mês sem aplicar (ORC-03/04).

**Response:**
```json
{
  "data": {
    "current_competence": "2026-06",
    "current_ceiling": 5000.00,
    "current_spending": 4200.00,
    "next_competence": "2026-07",
    "suggested_ceiling": 4200.00,
    "adjustment_reason": "spending_below_ceiling"
  }
}
```

### `POST /budgets/global/auto-adjust`
Aplicar auto-ajuste para a próxima competência (ORC-05).

**Body:**
```json
{ "user_id": "uuid", "reference_competence": "2026-06" }
```

---

## Saldo (SAL)

### `GET /balance?user_id=&competence=`
Retorna o saldo completo do usuário na competência.

**Response:**
```json
{
  "data": {
    "competence": "2026-06",
    "total_income": 7500.00,
    "total_expenses": 3200.00,
    "balance_today": 4300.00,
    "committed_bills": 850.00,
    "projected_balance": 3450.00,
    "global_ceiling": 5000.00,
    "ceiling_usage_pct": 64.0,
    "projected_ceiling_usage_pct": 81.0,
    "is_projected_negative": false
  }
}
```

---

## Metas de Redução (MET)

### `POST /goals/reduction`
Criar meta de redução para uma categoria no mês.

**Body:**
```json
{
  "user_id": "uuid",
  "category_id": "uuid",
  "competence": "2026-06",
  "target_amount": 400.00
}
```

### `GET /goals/reduction?user_id=&competence=`
Listar metas do mês.

### `PUT /goals/reduction/:id`
Editar meta.

### `DELETE /goals/reduction/:id?requester_id=`
Remover meta.

### `GET /goals/reduction/comparison?user_id=&competence=`
Comparativo: gasto atual vs. meta vs. mês anterior por categoria (MET-05/07).

**Response:**
```json
{
  "data": [
    {
      "category_id": "uuid",
      "category_name": "Alimentação",
      "previous_month_amount": 800.00,
      "current_month_amount": 620.00,
      "target_amount": 700.00,
      "on_track": true,
      "variation_pct": -22.5,
      "variation_label": "22,5% menor que o mês passado",
      "target_progress_pct": 88.6
    }
  ]
}
```
