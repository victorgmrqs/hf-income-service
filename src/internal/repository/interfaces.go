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
