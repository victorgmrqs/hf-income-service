package goal

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
)

// mockReductionGoalRepository é um mock manual de repository.ReductionGoalRepository
// para os testes unitários dos use cases. Cada campo func customiza o cenário.
type mockReductionGoalRepository struct {
	CreateFn                            func(ctx context.Context, goal *entity.ReductionGoal) error
	GetByIDFn                           func(ctx context.Context, id uuid.UUID) (*entity.ReductionGoal, error)
	ListByUserAndCompetenceFn           func(ctx context.Context, userID uuid.UUID, competence string) ([]entity.ReductionGoal, error)
	UpdateFn                            func(ctx context.Context, goal *entity.ReductionGoal) error
	DeleteFn                            func(ctx context.Context, id uuid.UUID) error
	ExistsByUserCategoryAndCompetenceFn func(ctx context.Context, userID, categoryID uuid.UUID, competence string) (bool, error)
	ListWithNullAchievedByCompetenceFn  func(ctx context.Context, competence string) ([]entity.ReductionGoal, error)
	UpdatePreviousAmountFn              func(ctx context.Context, id uuid.UUID, previousAmount decimal.Decimal) error
	SetAchievedFn                       func(ctx context.Context, id uuid.UUID, achieved bool) error
}

var _ repository.ReductionGoalRepository = (*mockReductionGoalRepository)(nil)

func (m *mockReductionGoalRepository) Create(ctx context.Context, goal *entity.ReductionGoal) error {
	return m.CreateFn(ctx, goal)
}

func (m *mockReductionGoalRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.ReductionGoal, error) {
	return m.GetByIDFn(ctx, id)
}

func (m *mockReductionGoalRepository) ListByUserAndCompetence(ctx context.Context, userID uuid.UUID, competence string) ([]entity.ReductionGoal, error) {
	return m.ListByUserAndCompetenceFn(ctx, userID, competence)
}

func (m *mockReductionGoalRepository) Update(ctx context.Context, goal *entity.ReductionGoal) error {
	return m.UpdateFn(ctx, goal)
}

func (m *mockReductionGoalRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.DeleteFn(ctx, id)
}

func (m *mockReductionGoalRepository) ExistsByUserCategoryAndCompetence(ctx context.Context, userID, categoryID uuid.UUID, competence string) (bool, error) {
	return m.ExistsByUserCategoryAndCompetenceFn(ctx, userID, categoryID, competence)
}

func (m *mockReductionGoalRepository) ListWithNullAchievedByCompetence(ctx context.Context, competence string) ([]entity.ReductionGoal, error) {
	return m.ListWithNullAchievedByCompetenceFn(ctx, competence)
}

func (m *mockReductionGoalRepository) UpdatePreviousAmount(ctx context.Context, id uuid.UUID, previousAmount decimal.Decimal) error {
	return m.UpdatePreviousAmountFn(ctx, id, previousAmount)
}

func (m *mockReductionGoalRepository) SetAchieved(ctx context.Context, id uuid.UUID, achieved bool) error {
	return m.SetAchievedFn(ctx, id, achieved)
}
