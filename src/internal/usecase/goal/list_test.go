package goal

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
)

func TestGoalList_Success(t *testing.T) {
	repo := &mockReductionGoalRepository{
		ListByUserAndCompetenceFn: func(context.Context, uuid.UUID, string) ([]entity.ReductionGoal, error) {
			return []entity.ReductionGoal{
				{ID: uuid.New(), Competence: "2026-06", TargetAmount: decimal.RequireFromString("400.00")},
				{ID: uuid.New(), Competence: "2026-06", TargetAmount: decimal.RequireFromString("250.00")},
			}, nil
		},
	}
	uc := NewListUseCase(repo, testLogger())

	out, err := uc.Execute(context.Background(), ListInput{UserID: uuid.New(), Competence: "2026-06"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("items = %d, want 2", len(out))
	}
}

func TestGoalList_Empty(t *testing.T) {
	repo := &mockReductionGoalRepository{
		ListByUserAndCompetenceFn: func(context.Context, uuid.UUID, string) ([]entity.ReductionGoal, error) {
			return []entity.ReductionGoal{}, nil
		},
	}
	uc := NewListUseCase(repo, testLogger())

	out, err := uc.Execute(context.Background(), ListInput{UserID: uuid.New(), Competence: "2026-06"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("items = %d, want 0", len(out))
	}
}

func TestGoalList_MissingRequiredField(t *testing.T) {
	uc := NewListUseCase(&mockReductionGoalRepository{}, testLogger())

	_, err := uc.Execute(context.Background(), ListInput{UserID: uuid.Nil, Competence: "2026-06"})
	if !errors.Is(err, ErrMissingRequiredField) {
		t.Fatalf("err = %v, want ErrMissingRequiredField", err)
	}
}

func TestGoalList_InvalidCompetence(t *testing.T) {
	uc := NewListUseCase(&mockReductionGoalRepository{}, testLogger())

	_, err := uc.Execute(context.Background(), ListInput{UserID: uuid.New(), Competence: "junho"})
	if !errors.Is(err, ErrInvalidCompetence) {
		t.Fatalf("err = %v, want ErrInvalidCompetence", err)
	}
}

// TestGoalList_RepositoryError cobre a propagação de um erro técnico do repositório.
func TestGoalList_RepositoryError(t *testing.T) {
	dbErr := errors.New("db unavailable")
	repo := &mockReductionGoalRepository{
		ListByUserAndCompetenceFn: func(context.Context, uuid.UUID, string) ([]entity.ReductionGoal, error) {
			return nil, dbErr
		},
	}
	uc := NewListUseCase(repo, testLogger())

	_, err := uc.Execute(context.Background(), ListInput{UserID: uuid.New(), Competence: "2026-06"})
	if !errors.Is(err, dbErr) {
		t.Fatalf("err = %v, want dbErr propagated", err)
	}
}
