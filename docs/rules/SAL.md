# SAL — Saldo Mensal

> Fonte de verdade: Notion. Este arquivo é espelho local — em caso de divergência, o Notion prevalece.
> Saldo não é persistido — é calculado sob demanda agregando dados deste serviço e do `hf-transaction-service`.
> Ver contrato de integração: [`INTEGRATIONS.md`](../../INTEGRATIONS.md)

| ID | Regra | Status | Critério de Aceite |
|----|-------|--------|-------------------|
| SAL-01 | Saldo atual = receita total da competência − (despesas pessoais + parte do usuário nas compartilhadas) até hoje | 📅 | `balance_today` = `total_income` − `total_expenses` (obtido via hf-transaction-service) |
| SAL-02 | Despesas comprometidas = contas a pagar com `status = PENDING` e `due_date` dentro da competência atual | 📅 | `committed_bills` = soma dos `amount` das contas pendentes no mês |
| SAL-03 | Saldo projetado = saldo atual − despesas comprometidas ainda não pagas | 📅 | `projected_balance` = `balance_today` − `committed_bills` |
| SAL-04 | Response de `/balance` exibe: `balance_today` e `projected_balance` | 📅 | Ambos os campos sempre presentes no response, mesmo que zero |
| SAL-05 | Se `projected_balance` < 0, exibir alerta visual (`is_projected_negative: true`) | 📅 | Campo `is_projected_negative` no response indica estado ao frontend |
