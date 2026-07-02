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

func TestCreateGlobalBudget_Success(t *testing.T) {
	repo := &mockGlobalBudgetRepository{
		GetByUserAndCompetenceFn: func(context.Context, uuid.UUID, string) (*entity.GlobalBudget, error) {
			return nil, gorm.ErrRecordNotFound
		},
		CreateFn: func(_ context.Context, budget *entity.GlobalBudget) error {
			budget.ID = uuid.New()
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
	if !out.Ceiling.Equal(in.Ceiling) {
		t.Errorf("ceiling = %s, want %s", out.Ceiling, in.Ceiling)
	}
	if out.AutoAdjusted {
		t.Error("auto_adjusted = true, want false on create")
	}
}

func TestCreateGlobalBudget_MissingRequiredField(t *testing.T) {
	repo := &mockGlobalBudgetRepository{
		CreateFn: func(context.Context, *entity.GlobalBudget) error {
			t.Fatal("repo.Create should not be called on validation failure")
			return nil
		},
	}
	uc := NewCreateUseCase(repo, testLogger())

	in := validCreateInput()
	in.Competence = ""
	_, err := uc.Execute(context.Background(), in)
	if !errors.Is(err, ErrMissingRequiredField) {
		t.Fatalf("err = %v, want ErrMissingRequiredField", err)
	}
}

func TestCreateGlobalBudget_InvalidCeiling(t *testing.T) {
	uc := NewCreateUseCase(&mockGlobalBudgetRepository{}, testLogger())

	in := validCreateInput()
	in.Ceiling = decimal.Zero
	_, err := uc.Execute(context.Background(), in)
	if !errors.Is(err, ErrInvalidCeiling) {
		t.Fatalf("err = %v, want ErrInvalidCeiling", err)
	}
}

func TestCreateGlobalBudget_InvalidCompetence(t *testing.T) {
	uc := NewCreateUseCase(&mockGlobalBudgetRepository{}, testLogger())

	in := validCreateInput()
	in.Competence = "2026/06"
	_, err := uc.Execute(context.Background(), in)
	if !errors.Is(err, ErrInvalidCompetence) {
		t.Fatalf("err = %v, want ErrInvalidCompetence", err)
	}
}

func TestCreateGlobalBudget_BudgetAlreadyExists(t *testing.T) {
	repo := &mockGlobalBudgetRepository{
		GetByUserAndCompetenceFn: func(_ context.Context, userID uuid.UUID, competence string) (*entity.GlobalBudget, error) {
			return &entity.GlobalBudget{ID: uuid.New(), UserID: userID, Competence: competence}, nil
		},
		CreateFn: func(context.Context, *entity.GlobalBudget) error {
			t.Fatal("repo.Create should not be called when budget already exists")
			return nil
		},
	}
	uc := NewCreateUseCase(repo, testLogger())

	_, err := uc.Execute(context.Background(), validCreateInput())
	if !errors.Is(err, ErrBudgetAlreadyExists) {
		t.Fatalf("err = %v, want ErrBudgetAlreadyExists", err)
	}
}
