package repository

import (
	"context"

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
