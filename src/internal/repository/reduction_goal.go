package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
	"gorm.io/gorm"
)

type reductionGoalRepository struct {
	db *gorm.DB
}

func NewReductionGoalRepository(db *gorm.DB) ReductionGoalRepository {
	return &reductionGoalRepository{db: db}
}

// EnsureReductionGoalIndexes aplica os índices parciais de MET-04 (FDD-003 §8):
// (user_id, category_id, competence) UNIQUE e (user_id, competence), ambos
// WHERE deleted_at IS NULL — a tag do GORM não expressa índice parcial, e um
// índice full impediria recriar a meta após soft delete (mesmo padrão do
// ADR-001/EnsureGlobalBudgetIndexes). Remove os índices full do HF-59 (drop
// idempotente; no-op a partir da segunda execução). Hoje chamado apenas no setup
// dos testes de integração (entity_integration_test.go); o wiring no main.go
// (AutoMigrate de ReductionGoal + esta função no boot) fica para a task que liga
// o domínio MET ao servidor (T14+), quando o repositório passa a ser consumido.
func EnsureReductionGoalIndexes(db *gorm.DB) error {
	if err := db.Exec(`DROP INDEX IF EXISTS idx_reduction_goals_user_category_competence`).Error; err != nil {
		return err
	}
	if err := db.Exec(`DROP INDEX IF EXISTS idx_reduction_goals_user_competence`).Error; err != nil {
		return err
	}
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_reduction_goals_user_category_competence_active
		ON reduction_goals (user_id, category_id, competence) WHERE deleted_at IS NULL`).Error; err != nil {
		return err
	}
	return db.Exec(`CREATE INDEX IF NOT EXISTS idx_reduction_goals_user_competence_active
		ON reduction_goals (user_id, competence) WHERE deleted_at IS NULL`).Error
}

func (r *reductionGoalRepository) Create(ctx context.Context, goal *entity.ReductionGoal) error {
	return r.db.WithContext(ctx).Create(goal).Error
}

// GetByID retorna gorm.ErrRecordNotFound quando o registro não existe ou está
// soft-deleted (a cláusula deleted_at IS NULL é aplicada automaticamente pelo GORM).
func (r *reductionGoalRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.ReductionGoal, error) {
	var goal entity.ReductionGoal
	if err := r.db.WithContext(ctx).First(&goal, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &goal, nil
}

func (r *reductionGoalRepository) ListByUserAndCompetence(ctx context.Context, userID uuid.UUID, competence string) ([]entity.ReductionGoal, error) {
	var goals []entity.ReductionGoal
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND competence = ?", userID, competence).
		Order("created_at ASC").
		Find(&goals).Error
	if err != nil {
		return nil, err
	}
	return goals, nil
}

// Update persiste todos os campos do registro (Save). user_id, category_id e
// competence são mantidos imutáveis pela camada de use case — apenas
// target_amount é editável (FDD-003 §5).
func (r *reductionGoalRepository) Update(ctx context.Context, goal *entity.ReductionGoal) error {
	return r.db.WithContext(ctx).Save(goal).Error
}

// Delete aplica soft delete (gorm.DeletedAt). Retorna ErrRecordNotFound quando
// nenhum registro ativo é afetado.
func (r *reductionGoalRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&entity.ReductionGoal{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ExistsByUserCategoryAndCompetence indica se há meta ativa para o trio (MET-04).
// Registros soft-deletados são ignorados automaticamente pelo GORM.
func (r *reductionGoalRepository) ExistsByUserCategoryAndCompetence(ctx context.Context, userID, categoryID uuid.UUID, competence string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.ReductionGoal{}).
		Where("user_id = ? AND category_id = ? AND competence = ?", userID, categoryID, competence).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListWithNullAchievedByCompetence retorna as metas ainda não fechadas da
// competência — base do job de fechamento do mês (MET-06).
func (r *reductionGoalRepository) ListWithNullAchievedByCompetence(ctx context.Context, competence string) ([]entity.ReductionGoal, error) {
	var goals []entity.ReductionGoal
	err := r.db.WithContext(ctx).
		Where("competence = ? AND achieved IS NULL", competence).
		Order("created_at ASC").
		Find(&goals).Error
	if err != nil {
		return nil, err
	}
	return goals, nil
}

// UpdatePreviousAmount preenche o snapshot retroativo do gasto do mês anterior
// obtido no comparativo (FDD-003 §4). Retorna ErrRecordNotFound quando nenhum
// registro ativo é afetado.
func (r *reductionGoalRepository) UpdatePreviousAmount(ctx context.Context, id uuid.UUID, previousAmount decimal.Decimal) error {
	res := r.db.WithContext(ctx).Model(&entity.ReductionGoal{}).
		Where("id = ?", id).
		Update("previous_amount", previousAmount)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// SetAchieved aplica o resultado do fechamento do mês (MET-06). Retorna
// ErrRecordNotFound quando nenhum registro ativo é afetado.
func (r *reductionGoalRepository) SetAchieved(ctx context.Context, id uuid.UUID, achieved bool) error {
	res := r.db.WithContext(ctx).Model(&entity.ReductionGoal{}).
		Where("id = ?", id).
		Update("achieved", achieved)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
