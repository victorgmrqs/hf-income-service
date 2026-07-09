# ADR-001 — Soft delete em GlobalBudget com índice único parcial

- **Status:** Aceito
- **Data:** 2026-07-02
- **Tickets:** HF-44 (decisão) · HF-59 (decisão superseded) · HF-64 (AC de origem)

## Contexto

O HF-59 criou a entidade `GlobalBudget` **sem** soft delete ("substituição direta ao editar"),
com índice único full em `(user_id, competence)` via tag GORM (ORC-01). O AC do HF-64
(absorvido pelo HF-44) exigia `Delete` soft — conflito de decisões. Na consolidação do
HF-44 (2026-07-02) foi decidido **manter o soft delete**, alinhando `GlobalBudget` às
demais entidades do serviço (`Income`, `ReductionGoal`).

## Problema

Com soft delete, um índice único full em `(user_id, competence)` **bloqueia a recriação**
do teto de uma competência já excluída: a linha soft-deletada continua ocupando o índice,
e o `INSERT` seguinte viola a constraint — quebrando o fluxo "deletou → recria o teto do mês".

## Decisão

1. `GlobalBudget` ganha `DeletedAt gorm.DeletedAt` (queries filtram `deleted_at IS NULL`
   automaticamente).
2. A unicidade de ORC-01 passa a ser um **índice único parcial**:
   `CREATE UNIQUE INDEX ... ON global_budgets (user_id, competence) WHERE deleted_at IS NULL`.
3. Como a tag `uniqueIndex` do GORM não expressa índice parcial, a migração é aplicada
   **fora do AutoMigrate** por `repository.EnsureGlobalBudgetIndexes(db)` — chamada no
   `main.go` e nos setups de teste de integração. A função também dropa o índice full
   antigo do HF-59 (`DROP INDEX IF EXISTS`, idempotente).
4. Exclusão disponível **apenas na camada de repositório** (`Delete`); não há endpoint
   `DELETE /budgets/global/:id` — será criado somente se um ticket futuro o exigir.

## Consequências

- (+) Recriar o teto de uma competência após exclusão funciona; auditoria preservada.
- (+) Consistência com a estratégia de exclusão das demais entidades.
- (−) Unicidade vale só entre registros ativos — consultas manuais no banco devem
  considerar `deleted_at`.
- (−) Migração de índice fora do AutoMigrate: novos ambientes dependem de
  `EnsureGlobalBudgetIndexes` (falha de boot se o SQL não aplicar).

## Referências

- `ENTITIES.md` (campos e estratégia de exclusão) · `docs/fdds/fdd-002-orc.md` §8
- `src/internal/repository/global_budget.go` (`EnsureGlobalBudgetIndexes`)
- Teste: `TestGlobalBudgetRepository_SoftDelete_HiddenAndRecreatable`
