package income

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
)

func TestListIncome_ReturnsItemsAndTotal(t *testing.T) {
	repo := &mockIncomeRepository{
		ListFn: func(context.Context, uuid.UUID, string) ([]entity.Income, error) {
			return []entity.Income{
				{ID: uuid.New(), Amount: decimal.RequireFromString("5000.00"), Competence: "2026-06", Type: entity.IncomeTypeSalary},
				{ID: uuid.New(), Amount: decimal.RequireFromString("2500.00"), Competence: "2026-06", Type: entity.IncomeTypeFreelance},
			}, nil
		},
	}
	uc := NewListUseCase(repo, testLogger())

	out, err := uc.Execute(context.Background(), ListInput{UserID: uuid.New(), Competence: "2026-06"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(out.Items))
	}
	if !out.TotalIncome.Equal(decimal.RequireFromString("7500.00")) {
		t.Errorf("total_income = %s, want 7500.00", out.TotalIncome)
	}
}

func TestListIncome_InvalidCompetence(t *testing.T) {
	uc := NewListUseCase(&mockIncomeRepository{}, testLogger())

	_, err := uc.Execute(context.Background(), ListInput{UserID: uuid.New(), Competence: "junho"})
	if !errors.Is(err, ErrInvalidCompetence) {
		t.Fatalf("err = %v, want ErrInvalidCompetence", err)
	}
}
