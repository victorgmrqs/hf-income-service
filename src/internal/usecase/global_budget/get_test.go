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

func TestGetGlobalBudget_Success(t *testing.T) {
	userID := uuid.New()
	budgetID := uuid.New()
	repo := &mockGlobalBudgetRepository{
		GetByUserAndCompetenceFn: func(_ context.Context, uid uuid.UUID, competence string) (*entity.GlobalBudget, error) {
			return &entity.GlobalBudget{
				ID:         budgetID,
				UserID:     uid,
				Competence: competence,
				Ceiling:    decimal.RequireFromString("5000.00"),
			}, nil
		},
	}
	uc := NewGetUseCase(repo, testLogger())

	out, err := uc.Execute(context.Background(), GetInput{UserID: userID, Competence: "2026-06"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ID != budgetID {
		t.Errorf("id = %s, want %s", out.ID, budgetID)
	}
}

func TestGetGlobalBudget_BudgetNotFound(t *testing.T) {
	repo := &mockGlobalBudgetRepository{
		GetByUserAndCompetenceFn: func(context.Context, uuid.UUID, string) (*entity.GlobalBudget, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}
	uc := NewGetUseCase(repo, testLogger())

	_, err := uc.Execute(context.Background(), GetInput{UserID: uuid.New(), Competence: "2026-06"})
	if !errors.Is(err, ErrBudgetNotFound) {
		t.Fatalf("err = %v, want ErrBudgetNotFound", err)
	}
}
