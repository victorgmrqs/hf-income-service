package income

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
)

func TestCreateIncome_Success(t *testing.T) {
	repo := &mockIncomeRepository{
		CreateFn: func(_ context.Context, income *entity.Income) error {
			income.ID = uuid.New()
			return nil
		},
	}
	uc := NewCreateUseCase(repo, testLogger())

	in := validCreateInput()
	out, err := uc.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil || out.ID == uuid.Nil {
		t.Fatal("expected output with generated ID")
	}
	if !out.Amount.Equal(in.Amount) {
		t.Errorf("amount = %s, want %s", out.Amount, in.Amount)
	}
	if out.Type != string(entity.IncomeTypeSalary) {
		t.Errorf("type = %s, want SALARY", out.Type)
	}
}

func TestCreateIncome_MissingRequiredField(t *testing.T) {
	repo := &mockIncomeRepository{
		CreateFn: func(context.Context, *entity.Income) error {
			t.Fatal("repo.Create should not be called on validation failure")
			return nil
		},
	}
	uc := NewCreateUseCase(repo, testLogger())

	in := validCreateInput()
	in.Description = ""
	_, err := uc.Execute(context.Background(), in)
	if !errors.Is(err, ErrMissingRequiredField) {
		t.Fatalf("err = %v, want ErrMissingRequiredField", err)
	}
}

func TestCreateIncome_InvalidAmount(t *testing.T) {
	uc := NewCreateUseCase(&mockIncomeRepository{}, testLogger())

	in := validCreateInput()
	in.Amount = decimal.Zero
	_, err := uc.Execute(context.Background(), in)
	if !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("err = %v, want ErrInvalidAmount", err)
	}
}

func TestCreateIncome_InvalidIncomeType(t *testing.T) {
	uc := NewCreateUseCase(&mockIncomeRepository{}, testLogger())

	in := validCreateInput()
	in.Type = entity.IncomeType("CRYPTO")
	_, err := uc.Execute(context.Background(), in)
	if !errors.Is(err, ErrInvalidIncomeType) {
		t.Fatalf("err = %v, want ErrInvalidIncomeType", err)
	}
}

func TestCreateIncome_InvalidCompetence(t *testing.T) {
	uc := NewCreateUseCase(&mockIncomeRepository{}, testLogger())

	in := validCreateInput()
	in.Competence = "2026/06"
	_, err := uc.Execute(context.Background(), in)
	if !errors.Is(err, ErrInvalidCompetence) {
		t.Fatalf("err = %v, want ErrInvalidCompetence", err)
	}
}
