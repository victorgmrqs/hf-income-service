package goal

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
)

func TestGoalUpdate_Success_OnlyTargetAmount(t *testing.T) {
	goalID := uuid.New()
	userID := uuid.New()
	categoryID := uuid.New()
	previous := decimal.RequireFromString("520.00")
	achieved := false
	existing := &entity.ReductionGoal{
		ID:             goalID,
		UserID:         userID,
		CategoryID:     categoryID,
		Competence:     "2026-06",
		TargetAmount:   decimal.RequireFromString("400.00"),
		PreviousAmount: &previous,
		Achieved:       &achieved,
	}
	var updated *entity.ReductionGoal
	repo := &mockReductionGoalRepository{
		GetByIDFn: func(context.Context, uuid.UUID) (*entity.ReductionGoal, error) {
			return existing, nil
		},
		UpdateFn: func(_ context.Context, goal *entity.ReductionGoal) error {
			updated = goal
			return nil
		},
	}
	uc := NewUpdateUseCase(repo, testLogger())

	out, err := uc.Execute(context.Background(), UpdateInput{ID: goalID, TargetAmount: decimal.RequireFromString("350.00")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.TargetAmount.Equal(decimal.RequireFromString("350.00")) {
		t.Errorf("target_amount = %s, want 350.00", out.TargetAmount)
	}
	// Campos imutáveis devem permanecer intactos (MET-04).
	if updated.UserID != userID || updated.CategoryID != categoryID || updated.Competence != "2026-06" {
		t.Errorf("campos imutáveis alterados: user_id=%s category_id=%s competence=%s", updated.UserID, updated.CategoryID, updated.Competence)
	}
	if updated.PreviousAmount == nil || !updated.PreviousAmount.Equal(previous) {
		t.Errorf("previous_amount alterado: %v, want %s", updated.PreviousAmount, previous)
	}
	if updated.Achieved == nil || *updated.Achieved != achieved {
		t.Errorf("achieved alterado: %v, want %v", updated.Achieved, achieved)
	}
}

func TestGoalUpdate_NotFound(t *testing.T) {
	repo := &mockReductionGoalRepository{
		GetByIDFn: func(context.Context, uuid.UUID) (*entity.ReductionGoal, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}
	uc := NewUpdateUseCase(repo, testLogger())

	_, err := uc.Execute(context.Background(), UpdateInput{ID: uuid.New(), TargetAmount: decimal.RequireFromString("100.00")})
	if !errors.Is(err, ErrGoalNotFound) {
		t.Fatalf("err = %v, want ErrGoalNotFound", err)
	}
}

func TestGoalUpdate_InvalidTargetAmount(t *testing.T) {
	repo := &mockReductionGoalRepository{
		GetByIDFn: func(context.Context, uuid.UUID) (*entity.ReductionGoal, error) {
			return &entity.ReductionGoal{ID: uuid.New(), TargetAmount: decimal.RequireFromString("400.00")}, nil
		},
	}
	uc := NewUpdateUseCase(repo, testLogger())

	_, err := uc.Execute(context.Background(), UpdateInput{ID: uuid.New(), TargetAmount: decimal.Zero})
	if !errors.Is(err, ErrInvalidTargetAmount) {
		t.Fatalf("err = %v, want ErrInvalidTargetAmount", err)
	}
}

// TestGoalUpdate_RepositoryError cobre um erro técnico do GetByID distinto de
// gorm.ErrRecordNotFound — deve propagar o erro cru, não ErrGoalNotFound.
func TestGoalUpdate_RepositoryError(t *testing.T) {
	dbErr := errors.New("db unavailable")
	repo := &mockReductionGoalRepository{
		GetByIDFn: func(context.Context, uuid.UUID) (*entity.ReductionGoal, error) {
			return nil, dbErr
		},
	}
	uc := NewUpdateUseCase(repo, testLogger())

	_, err := uc.Execute(context.Background(), UpdateInput{ID: uuid.New(), TargetAmount: decimal.RequireFromString("100.00")})
	if !errors.Is(err, dbErr) {
		t.Fatalf("err = %v, want dbErr propagated", err)
	}
	if errors.Is(err, ErrGoalNotFound) {
		t.Error("technical error should not be mapped to ErrGoalNotFound")
	}
}
