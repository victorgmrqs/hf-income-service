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

	// ListRecurrentByCompetence retorna as receitas recorrentes de uma competência (REC-04).
	ListRecurrentByCompetence(ctx context.Context, competence string) ([]entity.Income, error)
	// ExistsByOriginAndCompetence indica se já existe uma cópia propagada (idempotência, REC-04).
	ExistsByOriginAndCompetence(ctx context.Context, originID uuid.UUID, competence string) (bool, error)
}

// GlobalBudgetRepository abstrai o acesso ao teto mensal de gastos (ORC).
// ORC-01: unicidade (user_id, competence) garantida por índice único parcial
// (WHERE deleted_at IS NULL — ver EnsureGlobalBudgetIndexes); soft delete (HF-44).
type GlobalBudgetRepository interface {
	Create(ctx context.Context, budget *entity.GlobalBudget) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.GlobalBudget, error)
	GetByUserAndCompetence(ctx context.Context, userID uuid.UUID, competence string) (*entity.GlobalBudget, error)
	Update(ctx context.Context, budget *entity.GlobalBudget) error

	// ExistsByUserAndCompetence indica se já existe teto ativo para o par (ORC-01).
	ExistsByUserAndCompetence(ctx context.Context, userID uuid.UUID, competence string) (bool, error)
	// Upsert cria o teto se não existir e atualiza ceiling/auto_adjusted se existir
	// (idempotência do auto-ajuste — ORC-03/04/05).
	Upsert(ctx context.Context, budget *entity.GlobalBudget) error
	// Delete aplica soft delete (gorm.DeletedAt).
	Delete(ctx context.Context, id uuid.UUID) error
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
