# Regras de Negócio — hf-income-service

> Fonte de verdade: Notion — 📏 Regras de Negócio.
> Este arquivo é um espelho local. Em caso de divergência, o Notion prevalece.
> Convenção de ID: `DOMINIO-NN`
> Status: ✅ Implementado | 🚧 Parcial | 📅 Planejado

---

## REC — Receitas

| ID | Descrição | Status |
|----|-----------|--------|
| REC-01 | Toda receita tem: descrição, valor, data, competência, tipo (SALARY \| FREELANCE \| INVESTMENT \| RENTAL \| OTHER), user_id | 📅 |
| REC-02 | Valor da receita deve ser > 0 | 📅 |
| REC-03 | Receita pode ser marcada como recorrente mensal (`recurrent: true`); edições se aplicam a partir da competência atual — histórico não é alterado | 📅 |
| REC-04 | Receitas recorrentes são propagadas para a próxima competência via job agendado (`POST /income/propagate`) | 📅 |
| REC-05 | Receita individual por padrão; receita compartilhada (fundo familiar) é funcionalidade futura | 📅 |
| REC-06 | A soma de todas as receitas de uma competência para um usuário é a **receita total do mês** | 📅 |

---

## ORC — Orçamento Global

| ID | Descrição | Status |
|----|-----------|--------|
| ORC-01 | Teto global é um valor mensal definido por usuário por competência | 📅 |
| ORC-02 | Teto padrão inicial pode ser igual à receita total do mês (opcional) | 📅 |
| ORC-03 | **Auto-ajuste:** se gasto do mês anterior < teto anterior → próximo teto = gasto anterior (aperto progressivo) | 📅 |
| ORC-04 | Se gasto anterior = teto anterior → próximo teto permanece igual | 📅 |
| ORC-05 | Auto-ajuste ocorre no 1º dia do mês; o teto é sempre editável manualmente pelo usuário | 📅 |
| ORC-06 | Dashboard alerta se soma dos orçamentos por categoria ultrapassar o teto global | 📅 |
| ORC-07 | Progresso global = total gasto no mês ÷ teto global | 📅 |

---

## SAL — Saldo Mensal

| ID | Descrição | Status |
|----|-----------|--------|
| SAL-01 | Saldo atual = receita total da competência − (despesas pessoais + minha parte das compartilhadas) até hoje | 📅 |
| SAL-02 | Despesas comprometidas = contas a pagar com `status = PENDING` e `due_date` dentro da competência atual | 📅 |
| SAL-03 | Saldo projetado = saldo atual − despesas comprometidas ainda não pagas no mês | 📅 |
| SAL-04 | Dashboard exibe: saldo hoje e saldo projetado considerando contas a pagar | 📅 |
| SAL-05 | Se saldo projetado for negativo, exibir alerta visual | 📅 |

---

## MET — Metas de Redução *(extensão do hf-transaction-service)*

| ID | Descrição | Status |
|----|-----------|--------|
| MET-04 | Usuário pode definir uma meta de redução por categoria em valor absoluto (ex: R$400 em Alimentação) | 📅 |
| MET-05 | Comparativo mensal: exibe gasto atual vs. meta vs. gasto do mês anterior com % de variação | 📅 |
| MET-06 | Ao final do mês, o sistema registra se a meta foi atingida (`achieved: bool`) | 📅 |
| MET-07 | Label de variação: "X% menor/maior que o mês passado" | 📅 |

---

## Glossário

| Termo | Definição |
|-------|-----------|
| Competência | Período de referência no formato `YYYY-MM` (≠ data do lançamento) |
| Teto global | Limite máximo de gastos totais do mês definido pelo usuário |
| Receita recorrente | Receita que se repete automaticamente todo mês até ser cancelada ou editada |
| Saldo projetado | Saldo atual descontadas as despesas certas ainda não pagas no mês |
| Meta de redução | Valor absoluto que o usuário se compromete a não ultrapassar em uma categoria |
