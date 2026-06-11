# REC — Receitas

> Fonte de verdade: Notion. Este arquivo é espelho local — em caso de divergência, o Notion prevalece.

| ID | Regra | Status | Critério de Aceite |
|----|-------|--------|-------------------|
| REC-01 | Toda receita tem: `description`, `amount`, `date`, `competence`, `type`, `user_id` | 📅 | Requisição sem qualquer campo obrigatório retorna `400` |
| REC-02 | `amount` deve ser > 0 | 📅 | Valor zero ou negativo retorna `400` com code `INVALID_AMOUNT` |
| REC-03 | Receita pode ser `recurrent: true`; edições valem a partir da competência atual — histórico preservado | 📅 | Editar receita recorrente não altera registros de competências anteriores |
| REC-04 | Receitas recorrentes são propagadas para a próxima competência via `POST /income/propagate` | 📅 | Job executado no 1º dia do mês cria cópia com `origin_id` apontando para a receita original |
| REC-05 | Receita é individual por padrão; receita compartilhada (fundo familiar) é funcionalidade futura | 📅 | — |
| REC-06 | Receita total do mês = soma de todas as receitas do `user_id` na `competence` | 📅 | Endpoint `/balance` usa esse valor como `total_income` |

**Tipos válidos:** `SALARY` \| `FREELANCE` \| `INVESTMENT` \| `RENTAL` \| `OTHER`
