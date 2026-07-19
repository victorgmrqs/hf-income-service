package global_budget

import (
	"context"

	"github.com/google/uuid"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
)

// mockGlobalBudgetRepository é um mock manual de repository.GlobalBudgetRepository
// para os testes unitários dos use cases. Cada campo func customiza o cenário.
type mockGlobalBudgetRepository struct {
	CreateFn                 func(ctx context.Context, budget *entity.GlobalBudget) error
	GetByIDFn                func(ctx context.Context, id uuid.UUID) (*entity.GlobalBudget, error)
	GetByUserAndCompetenceFn func(ctx context.Context, userID uuid.UUID, competence string) (*entity.GlobalBudget, error)
	UpdateFn                 func(ctx context.Context, budget *entity.GlobalBudget) error
	ExistsFn                 func(ctx context.Context, userID uuid.UUID, competence string) (bool, error)
	UpsertFn                 func(ctx context.Context, budget *entity.GlobalBudget) error
	DeleteFn                 func(ctx context.Context, id uuid.UUID) error
	ListUserIDsFn            func(ctx context.Context, competence string) ([]uuid.UUID, error)
}

var _ repository.GlobalBudgetRepository = (*mockGlobalBudgetRepository)(nil)

func (m *mockGlobalBudgetRepository) Create(ctx context.Context, budget *entity.GlobalBudget) error {
	return m.CreateFn(ctx, budget)
}

func (m *mockGlobalBudgetRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.GlobalBudget, error) {
	return m.GetByIDFn(ctx, id)
}

func (m *mockGlobalBudgetRepository) GetByUserAndCompetence(ctx context.Context, userID uuid.UUID, competence string) (*entity.GlobalBudget, error) {
	return m.GetByUserAndCompetenceFn(ctx, userID, competence)
}

func (m *mockGlobalBudgetRepository) Update(ctx context.Context, budget *entity.GlobalBudget) error {
	return m.UpdateFn(ctx, budget)
}

func (m *mockGlobalBudgetRepository) ExistsByUserAndCompetence(ctx context.Context, userID uuid.UUID, competence string) (bool, error) {
	return m.ExistsFn(ctx, userID, competence)
}

func (m *mockGlobalBudgetRepository) Upsert(ctx context.Context, budget *entity.GlobalBudget) error {
	return m.UpsertFn(ctx, budget)
}

func (m *mockGlobalBudgetRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.DeleteFn(ctx, id)
}

func (m *mockGlobalBudgetRepository) ListUserIDsByCompetence(ctx context.Context, competence string) ([]uuid.UUID, error) {
	return m.ListUserIDsFn(ctx, competence)
}
