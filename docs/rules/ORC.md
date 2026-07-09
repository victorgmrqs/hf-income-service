# ORC — Orçamento Global

> Fonte de verdade: Notion. Este arquivo é espelho local — em caso de divergência, o Notion prevalece.

| ID | Regra | Status | Critério de Aceite |
|----|-------|--------|-------------------|
| ORC-01 | Teto global é um valor mensal definido por `user_id` e `competence` | ✅ | Cada usuário tem no máximo um teto por competência (unique `(user_id, competence)` + 409; `ceiling > 0`; edição manual força `auto_adjusted=false`) — HF-43 |
| ORC-02 | Teto padrão inicial pode ser igual à receita total do mês (opcional, definido pelo usuário) | 📅 | — |
| ORC-03 | **Auto-ajuste:** se gasto anterior < teto anterior → próximo teto = gasto anterior | ✅ | Resultado de `auto-adjust` é auditável: `auto_adjusted: true` no registro; sem teto anterior → `400 NO_PREVIOUS_BUDGET` — HF-44 |
| ORC-04 | Se gasto anterior >= teto anterior → próximo teto permanece igual | ✅ | `auto_adjusted: false` quando não há ajuste — HF-44 |
| ORC-05 | Auto-ajuste ocorre no 1º dia do mês via job (`POST /budgets/global/auto-adjust`, idempotente); teto é sempre editável manualmente | ✅ | Edição manual sobrescreve o auto-ajuste e define `auto_adjusted: false` — HF-44 |
| ORC-06 | Alerta quando soma dos orçamentos por categoria (via hf-transaction-service) ultrapassar o teto global | 📅 | Retornado no response de `/balance` como campo `ceiling_exceeded: bool` |
| ORC-07 | Progresso global = total gasto no mês ÷ teto global (retornado como `ceiling_usage_pct`) | 📅 | Valor entre 0 e 1; acima de 1 indica estouro |
