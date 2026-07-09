# Modelo de Dados — hf-income-service

## Diagrama de Relacionamentos

```
User (externo — hf-transaction-service)
  │
  ├── Income [1:N]          (receitas do usuário)
  ├── GlobalBudget [1:N]    (um teto por competência)
  └── ReductionGoal [1:N]   (metas por categoria/competência)

Income
  └── origin_id → Income    (auto-referência para recorrências)
```

> `User` é gerenciado pelo `hf-transaction-service`. Este serviço referencia `user_id` (UUID) sem FK direta — consistência eventual.

---

## 1. Income (Receita)

| Campo | Tipo | Obrigatório | Descrição |
|-------|------|-------------|-----------|
| id | UUID | Sim | Identificador único |
| user_id | UUID | Sim | Referência ao usuário |
| description | string(200) | Sim | Descrição da receita |
| amount | decimal(10,2) | Sim | Valor da receita (> 0) |
| type | enum | Sim | SALARY \| FREELANCE \| INVESTMENT \| RENTAL \| OTHER |
| date | date | Sim | Data do recebimento |
| competence | string(7) | Sim | Mês de referência (YYYY-MM) |
| recurrent | bool | Sim | Se deve ser repetida mensalmente |
| origin_id | UUID | Não | ID da receita-origem (se gerada por recorrência) |
| deleted_at | timestamp | Não | Soft delete (GORM) |
| created_at | timestamp | Sim | — |
| updated_at | timestamp | Sim | — |

**Constraints:**
- `amount` > 0 (REC-02)
- `competence` deve corresponder ao mês/ano de `date`
- Combinação `(user_id, competence, type, description)` não é unique — múltiplas fontes são permitidas

**Índices:**
- `(user_id, competence)` — listagem por mês
- `(origin_id)` — busca de ocorrências de recorrências

---

## 2. GlobalBudget (Teto Global)

| Campo | Tipo | Obrigatório | Descrição |
|-------|------|-------------|-----------|
| id | UUID | Sim | Identificador único |
| user_id | UUID | Sim | Referência ao usuário |
| competence | string(7) | Sim | Mês de referência (YYYY-MM) |
| ceiling | decimal(10,2) | Sim | Valor máximo de gastos do mês (> 0) |
| auto_adjusted | bool | Sim | `true` se definido pelo auto-ajuste; `false` se editado manualmente |
| created_at | timestamp | Sim | — |
| updated_at | timestamp | Sim | — |
| deleted_at | timestamp | Não | Soft delete (HF-44 — supersede o "sem soft delete" do HF-59; ver ADR-001) |

**Constraints:**
- `ceiling` > 0 (criação/edição manual; o auto-ajuste pode resultar em teto 0 quando o gasto anterior é 0 — FDD-002 §4)
- Combinação `(user_id, competence)` deve ser única **entre registros ativos** (um teto por mês por usuário — ORC-01)

**Índices:**
- `(user_id, competence)` UNIQUE **parcial** (`WHERE deleted_at IS NULL`) — criado por `repository.EnsureGlobalBudgetIndexes` fora do AutoMigrate (a tag do GORM não expressa índice parcial); permite recriar o teto da competência após soft delete (HF-44)

---

## 3. ReductionGoal (Meta de Redução)

| Campo | Tipo | Obrigatório | Descrição |
|-------|------|-------------|-----------|
| id | UUID | Sim | Identificador único |
| user_id | UUID | Sim | Referência ao usuário |
| category_id | UUID | Sim | Categoria (referência externa — hf-transaction-service) |
| competence | string(7) | Sim | Mês de referência (YYYY-MM) |
| target_amount | decimal(10,2) | Sim | Valor máximo que deseja gastar nesta categoria |
| previous_amount | decimal(10,2) | Não | Snapshot do gasto do mês anterior (capturado na criação; `null` quando hf-transaction-service indisponível) |
| achieved | bool | Não | `null` = mês em curso; `true/false` = resultado final |
| deleted_at | timestamp | Não | Soft delete (GORM) |
| created_at | timestamp | Sim | — |
| updated_at | timestamp | Sim | — |

**Constraints:**
- `target_amount` > 0
- Combinação `(user_id, category_id, competence)` deve ser única (uma meta por categoria por mês)

**Índices:**
- `(user_id, competence)` — listagem por mês
- `(user_id, category_id, competence)` UNIQUE

---

## Regras de Integridade

| Regra | Descrição |
|-------|-----------|
| REC-03 | Edição de receita recorrente aplica a partir da competência atual; registros anteriores preservados |
| ORC-03 | `GlobalBudget` para nova competência criado automaticamente no 1º do mês com base no mês anterior |
| MET-06 | `achieved` atualizado ao final da competência comparando `target_amount` com gasto real |

## Estratégia de Exclusão

- **Income:** soft delete (`deleted_at`). Receitas deletadas não entram no cálculo de saldo.
- **GlobalBudget:** soft delete (`deleted_at`) desde HF-44 (antes: substituição direta — HF-59). Exclusão disponível apenas na camada de repositório (sem endpoint DELETE); unicidade via índice parcial. Ver ADR-001.
- **ReductionGoal:** soft delete (`deleted_at`).
