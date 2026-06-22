package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
)

// IncomeRepository abstrai o acesso a dados de receitas (REC).
type IncomeRepository interface {
	Create(ctx context.Context, income *entity.Income) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Income, error)
	ListByUserAndCompetence(ctx context.Context, userID uuid.UUID, competence string) ([]entity.Income, error)
	Update(ctx context.Context, income *entity.Income) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// GlobalBudgetRepository abstrai o acesso ao teto mensal de gastos (ORC).
// ORC-01: unicidade (user_id, competence) é garantida na entidade; sem soft delete.
type GlobalBudgetRepository interface {
	Create(ctx context.Context, budget *entity.GlobalBudget) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.GlobalBudget, error)
	GetByUserAndCompetence(ctx context.Context, userID uuid.UUID, competence string) (*entity.GlobalBudget, error)
	Update(ctx context.Context, budget *entity.GlobalBudget) error
}

// ReductionGoalRepository abstrai o acesso a metas de redução por categoria (MET).
// MET-04: unicidade (user_id, category_id, competence) é garantida na entidade; soft delete.
type ReductionGoalRepository interface {
	Create(ctx context.Context, goal *entity.ReductionGoal) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.ReductionGoal, error)
	ListByUserAndCompetence(ctx context.Context, userID uuid.UUID, competence string) ([]entity.ReductionGoal, error)
	Update(ctx context.Context, goal *entity.ReductionGoal) error
	Delete(ctx context.Context, id uuid.UUID) error
}
