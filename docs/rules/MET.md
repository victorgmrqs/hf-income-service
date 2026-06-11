# MET — Metas de Redução

> Fonte de verdade: Notion. Este arquivo é espelho local — em caso de divergência, o Notion prevalece.
> Este domínio é uma extensão do `hf-transaction-service` (categorias de despesa são gerenciadas lá).

| ID | Regra | Status | Critério de Aceite |
|----|-------|--------|-------------------|
| MET-04 | Usuário define meta de redução por categoria em **valor absoluto** (ex: R$ 400 em Alimentação) | 📅 | `target_amount` > 0; categoria deve existir no hf-transaction-service |
| MET-05 | Comparativo mensal: gasto atual vs. meta vs. gasto do mês anterior com % de variação | 📅 | Response inclui `variation_pct` e `variation_label` (ex: "22.5% menor que o mês passado") |
| MET-06 | No fechamento do mês o sistema registra se a meta foi atingida (`achieved: bool`) | 📅 | `achieved: true` quando `current_month_amount` ≤ `target_amount` |
| MET-07 | Tela de budgets exibe por categoria: limite, gasto atual, comparativo e meta | 📅 | Endpoint `GET /goals/reduction/comparison` retorna todos os campos necessários para o card |

**Nota:** `category_id` é uma referência externa — não há FK no banco deste serviço. Validação de existência é feita na camada de use case via chamada ao hf-transaction-service (futuro).
