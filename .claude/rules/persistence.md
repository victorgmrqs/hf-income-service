---
paths:
  - "src/internal/repository/**/*.go"
  - "src/internal/entity/**/*.go"
---
# Persistence

- Every entity gets its primary key generated in a `BeforeCreate(tx *gorm.DB) (err error)` GORM hook: `if x.ID == uuid.Nil { x.ID = uuid.New() }`. Never generate the UUID at construction time in the usecase.
- Soft delete via embedded `DeletedAt gorm.DeletedAt \`gorm:"index"\`` is the default for every entity. If an entity intentionally has no soft delete (or the behavior changed), that decision must be documented in `docs/adr/` and referenced in a comment on the field/struct — see `ADR-001` for the precedent.
- Every repository method takes `ctx context.Context` as first argument and calls `r.db.WithContext(ctx)...` — never `r.db` bare.
- Repository methods return errors from GORM as-is (e.g. `gorm.ErrRecordNotFound`), never wrapped — the usecase layer is what translates them into domain sentinels.
- Unique/composite constraints are expressed as GORM struct tags (`uniqueIndex:idx_name,priority:N`) directly on the entity fields, not as raw SQL migrations. When a partial index is needed (e.g. unique only among non-deleted rows), add an explicit `Ensure<Entity>Indexes` helper — see `EnsureGlobalBudgetIndexes`.
- Monetary fields are always `decimal.Decimal` (`github.com/shopspring/decimal`) mapped to `decimal(12,2)` — never `float64`/`float32`.
- Aggregation queries (`SUM`, `COUNT`) use `COALESCE(..., 0)` so an empty result set returns a zero value instead of `NULL`/error.
- List queries apply an explicit `Order(...)` clause — never rely on unspecified DB ordering.

Example (real, `repository/income.go`):

```go
func (r *incomeRepository) SumByUserAndCompetence(ctx context.Context, userID uuid.UUID, competence string) (decimal.Decimal, error) {
    var sum decimal.Decimal
    err := r.db.WithContext(ctx).Model(&entity.Income{}).
        Where("user_id = ? AND competence = ?", userID, competence).
        Select("COALESCE(SUM(amount), 0)").Scan(&sum).Error
    ...
}
```
