package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
	"gorm.io/gorm"
)

type globalBudgetRepository struct {
	db *gorm.DB
}

func NewGlobalBudgetRepository(db *gorm.DB) GlobalBudgetRepository {
	return &globalBudgetRepository{db: db}
}

// EnsureGlobalBudgetIndexes aplica o índice único parcial de ORC-01:
// (user_id, competence) WHERE deleted_at IS NULL — um teto ativo por competência,
// permitindo recriar após soft delete (HF-44). Remove o índice full antigo do
// HF-59 (drop idempotente; no-op a partir da segunda execução). Chamado no
// main.go após o AutoMigrate e no setup dos testes de integração.
func EnsureGlobalBudgetIndexes(db *gorm.DB) error {
	if err := db.Exec(`DROP INDEX IF EXISTS idx_global_budgets_user_competence`).Error; err != nil {
		return err
	}
	return db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_global_budgets_user_competence_active
		ON global_budgets (user_id, competence) WHERE deleted_at IS NULL`).Error
}

func (r *globalBudgetRepository) Create(ctx context.Context, budget *entity.GlobalBudget) error {
	return r.db.WithContext(ctx).Create(budget).Error
}

// GetByID retorna gorm.ErrRecordNotFound quando o registro não existe.
// GlobalBudget não usa soft delete (ORC/ENTITIES.md).
func (r *globalBudgetRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.GlobalBudget, error) {
	var budget entity.GlobalBudget
	if err := r.db.WithContext(ctx).First(&budget, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &budget, nil
}

// GetByUserAndCompetence busca o teto único da competência (ORC-01).
// Retorna gorm.ErrRecordNotFound quando não existe teto para o par.
func (r *globalBudgetRepository) GetByUserAndCompetence(ctx context.Context, userID uuid.UUID, competence string) (*entity.GlobalBudget, error) {
	var budget entity.GlobalBudget
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND competence = ?", userID, competence).
		First(&budget).Error
	if err != nil {
		return nil, err
	}
	return &budget, nil
}

// Update persiste todos os campos do registro (Save). A competência e o user_id
// são mantidos imutáveis pela camada de use case (ORC-01).
func (r *globalBudgetRepository) Update(ctx context.Context, budget *entity.GlobalBudget) error {
	return r.db.WithContext(ctx).Save(budget).Error
}

// ExistsByUserAndCompetence indica se há teto ativo para o par (ORC-01).
// Registros soft-deletados são ignorados automaticamente pelo GORM.
func (r *globalBudgetRepository) ExistsByUserAndCompetence(ctx context.Context, userID uuid.UUID, competence string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.GlobalBudget{}).
		Where("user_id = ? AND competence = ?", userID, competence).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Upsert cria o teto se não existir para (user_id, competence) e atualiza
// ceiling/auto_adjusted se já existir — idempotência do auto-ajuste (ORC-03/04/05).
// Corrida entre a busca e o Create é coberta pelo índice único parcial.
func (r *globalBudgetRepository) Upsert(ctx context.Context, budget *entity.GlobalBudget) error {
	var existing entity.GlobalBudget
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND competence = ?", budget.UserID, budget.Competence).
		First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.WithContext(ctx).Create(budget).Error
	}
	if err != nil {
		return err
	}
	existing.Ceiling = budget.Ceiling
	existing.AutoAdjusted = budget.AutoAdjusted
	if err := r.db.WithContext(ctx).Save(&existing).Error; err != nil {
		return err
	}
	*budget = existing
	return nil
}

// ListUserIDsByCompetence retorna os user_ids distintos com teto ativo na
// competência — elegíveis ao auto-ajuste do mês seguinte (ORC-05, HF-38).
// Soft-deletados ficam de fora pelo escopo padrão do GORM.
func (r *globalBudgetRepository) ListUserIDsByCompetence(ctx context.Context, competence string) ([]uuid.UUID, error) {
	var userIDs []uuid.UUID
	err := r.db.WithContext(ctx).
		Model(&entity.GlobalBudget{}).
		Where("competence = ?", competence).
		Distinct().
		Pluck("user_id", &userIDs).Error
	if err != nil {
		return nil, err
	}
	return userIDs, nil
}

// Delete aplica soft delete (gorm.DeletedAt). Retorna ErrRecordNotFound quando
// nenhum registro ativo é afetado.
func (r *globalBudgetRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&entity.GlobalBudget{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
