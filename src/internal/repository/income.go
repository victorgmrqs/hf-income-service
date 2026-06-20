package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
	"gorm.io/gorm"
)

type incomeRepository struct {
	db *gorm.DB
}

func NewIncomeRepository(db *gorm.DB) IncomeRepository {
	return &incomeRepository{db: db}
}

func (r *incomeRepository) Create(ctx context.Context, income *entity.Income) error {
	return r.db.WithContext(ctx).Create(income).Error
}

// FindByID retorna gorm.ErrRecordNotFound quando o registro não existe ou está
// soft-deleted (a cláusula deleted_at IS NULL é aplicada automaticamente pelo GORM).
func (r *incomeRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Income, error) {
	var income entity.Income
	if err := r.db.WithContext(ctx).First(&income, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &income, nil
}

func (r *incomeRepository) ListByUserAndCompetence(ctx context.Context, userID uuid.UUID, competence string) ([]entity.Income, error) {
	var incomes []entity.Income
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND competence = ?", userID, competence).
		Order("date ASC").
		Find(&incomes).Error
	if err != nil {
		return nil, err
	}
	return incomes, nil
}

// Update persiste todos os campos do registro (Save). A competência e o origin_id
// são mantidos imutáveis pela camada de use case (REC-03).
func (r *incomeRepository) Update(ctx context.Context, income *entity.Income) error {
	return r.db.WithContext(ctx).Save(income).Error
}

// Delete aplica soft delete (gorm.DeletedAt). Retorna ErrRecordNotFound quando
// nenhum registro ativo é afetado.
func (r *incomeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&entity.Income{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *incomeRepository) ListRecurrentByCompetence(ctx context.Context, competence string) ([]entity.Income, error) {
	var incomes []entity.Income
	err := r.db.WithContext(ctx).
		Where("recurrent = ? AND competence = ?", true, competence).
		Order("date ASC").
		Find(&incomes).Error
	if err != nil {
		return nil, err
	}
	return incomes, nil
}

func (r *incomeRepository) ExistsByOriginAndCompetence(ctx context.Context, originID uuid.UUID, competence string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.Income{}).
		Where("origin_id = ? AND competence = ?", originID, competence).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
