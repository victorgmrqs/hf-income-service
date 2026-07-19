package income

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
)

// mockIncomeRepository é um mock manual de repository.IncomeRepository para os
// testes unitários dos use cases. Cada campo func permite customizar o cenário.
type mockIncomeRepository struct {
	CreateFn   func(ctx context.Context, income *entity.Income) error
	FindByIDFn func(ctx context.Context, id uuid.UUID) (*entity.Income, error)
	ListFn     func(ctx context.Context, userID uuid.UUID, competence string) ([]entity.Income, error)
	UpdateFn   func(ctx context.Context, income *entity.Income) error
	DeleteFn   func(ctx context.Context, id uuid.UUID) error

	ListRecurrentFn func(ctx context.Context, competence string) ([]entity.Income, error)
	ExistsFn        func(ctx context.Context, originID uuid.UUID, competence string) (bool, error)
	SumFn           func(ctx context.Context, userID uuid.UUID, competence string) (decimal.Decimal, error)
}

var _ repository.IncomeRepository = (*mockIncomeRepository)(nil)

func (m *mockIncomeRepository) Create(ctx context.Context, income *entity.Income) error {
	return m.CreateFn(ctx, income)
}

func (m *mockIncomeRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Income, error) {
	return m.FindByIDFn(ctx, id)
}

func (m *mockIncomeRepository) ListByUserAndCompetence(ctx context.Context, userID uuid.UUID, competence string) ([]entity.Income, error) {
	return m.ListFn(ctx, userID, competence)
}

func (m *mockIncomeRepository) Update(ctx context.Context, income *entity.Income) error {
	return m.UpdateFn(ctx, income)
}

func (m *mockIncomeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.DeleteFn(ctx, id)
}

func (m *mockIncomeRepository) ListRecurrentByCompetence(ctx context.Context, competence string) ([]entity.Income, error) {
	return m.ListRecurrentFn(ctx, competence)
}

func (m *mockIncomeRepository) ExistsByOriginAndCompetence(ctx context.Context, originID uuid.UUID, competence string) (bool, error) {
	return m.ExistsFn(ctx, originID, competence)
}

func (m *mockIncomeRepository) SumByUserAndCompetence(ctx context.Context, userID uuid.UUID, competence string) (decimal.Decimal, error) {
	return m.SumFn(ctx, userID, competence)
}
