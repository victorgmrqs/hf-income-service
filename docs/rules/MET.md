# MET — Metas de Redução

> Fonte de verdade: Notion. Este arquivo é espelho local — em caso de divergência, o Notion prevalece.
> Este domínio é uma extensão do `hf-transaction-service` (categorias de despesa são gerenciadas lá).

| ID | Regra | Status | Critério de Aceite |
|----|-------|--------|-------------------|
| MET-04 | Usuário define meta de redução por categoria em **valor absoluto** (ex: R$ 400 em Alimentação) | ✅ use case (HF-68/HF-71); rota HTTP no HF-70 | `target_amount` > 0; categoria deve existir no hf-transaction-service |
| MET-05 | Comparativo mensal: gasto atual vs. meta vs. gasto do mês anterior com % de variação | ✅ use case (HF-69); rota HTTP no HF-70 | Response inclui `variation_pct` e `variation_label` (ex: "22,5% menor que o mês passado"); upstream indisponível degrada campos para `null` |
| MET-06 | No fechamento do mês o sistema registra se a meta foi atingida (`achieved: bool`) | ✅ use case (HF-69); rota HTTP no HF-70 | `achieved: true` quando `current_month_amount` ≤ `target_amount`; upstream indisponível pula a meta e segue as demais |
| MET-07 | Tela de budgets exibe por categoria: limite, gasto atual, comparativo e meta | 📅 (endpoint no HF-70; UI no HF-47) | Endpoint `GET /goals/reduction/comparison` retorna todos os campos necessários para o card |

**Nota:** `category_id` é uma referência externa — não há FK no banco deste serviço. Validação de existência é feita na camada de use case via chamada ao hf-transaction-service (futuro).
