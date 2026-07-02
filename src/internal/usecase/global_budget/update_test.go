package global_budget

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
)

func TestUpdateGlobalBudget_Success(t *testing.T) {
	budgetID := uuid.New()
	repo := &mockGlobalBudgetRepository{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*entity.GlobalBudget, error) {
			// teto originado de auto-ajuste — a edição manual deve zerar a flag (ORC-05).
			return &entity.GlobalBudget{
				ID:           id,
				UserID:       uuid.New(),
				Competence:   "2026-06",
				Ceiling:      decimal.RequireFromString("5000.00"),
				AutoAdjusted: true,
			}, nil
		},
		UpdateFn: func(context.Context, *entity.GlobalBudget) error { return nil },
	}
	uc := NewUpdateUseCase(repo, testLogger())

	out, err := uc.Execute(context.Background(), UpdateInput{
		ID:      budgetID,
		Ceiling: decimal.RequireFromString("4800.00"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Ceiling.Equal(decimal.RequireFromString("4800.00")) {
		t.Errorf("ceiling = %s, want 4800.00", out.Ceiling)
	}
	if out.AutoAdjusted {
		t.Error("auto_adjusted = true, want false after manual edit (ORC-05)")
	}
}

func TestUpdateGlobalBudget_InvalidCeiling(t *testing.T) {
	repo := &mockGlobalBudgetRepository{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*entity.GlobalBudget, error) {
			return &entity.GlobalBudget{ID: id, Competence: "2026-06", Ceiling: decimal.RequireFromString("5000.00")}, nil
		},
		UpdateFn: func(context.Context, *entity.GlobalBudget) error {
			t.Fatal("repo.Update should not be called on validation failure")
			return nil
		},
	}
	uc := NewUpdateUseCase(repo, testLogger())

	_, err := uc.Execute(context.Background(), UpdateInput{ID: uuid.New(), Ceiling: decimal.Zero})
	if !errors.Is(err, ErrInvalidCeiling) {
		t.Fatalf("err = %v, want ErrInvalidCeiling", err)
	}
}

func TestUpdateGlobalBudget_BudgetNotFound(t *testing.T) {
	repo := &mockGlobalBudgetRepository{
		GetByIDFn: func(context.Context, uuid.UUID) (*entity.GlobalBudget, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}
	uc := NewUpdateUseCase(repo, testLogger())

	_, err := uc.Execute(context.Background(), UpdateInput{ID: uuid.New(), Ceiling: decimal.RequireFromString("100.00")})
	if !errors.Is(err, ErrBudgetNotFound) {
		t.Fatalf("err = %v, want ErrBudgetNotFound", err)
	}
}
