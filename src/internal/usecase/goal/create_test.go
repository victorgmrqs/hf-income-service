package goal

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
	"github.com/victorgmrqs/hf-income-service/src/pkg/httpclient"
)

func TestGoalCreate_Success(t *testing.T) {
	previous := decimal.RequireFromString("520.00")
	repo := &mockReductionGoalRepository{
		ExistsByUserCategoryAndCompetenceFn: func(context.Context, uuid.UUID, uuid.UUID, string) (bool, error) {
			return false, nil
		},
		CreateFn: func(_ context.Context, goal *entity.ReductionGoal) error {
			goal.ID = uuid.New()
			return nil
		},
	}
	client := &mockTransactionClient{
		GetExpensesByCategoryFn: func(context.Context, string, string, string) (*httpclient.ExpensesByCategoryOutput, error) {
			return &httpclient.ExpensesByCategoryOutput{Total: previous}, nil
		},
	}
	uc := NewCreateUseCase(repo, client, testLogger())

	in := validCreateInput()
	out, err := uc.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil || out.ID == uuid.Nil {
		t.Fatal("expected output with generated ID")
	}
	if !out.TargetAmount.Equal(in.TargetAmount) {
		t.Errorf("target_amount = %s, want %s", out.TargetAmount, in.TargetAmount)
	}
	if out.PreviousAmount == nil || !out.PreviousAmount.Equal(previous) {
		t.Errorf("previous_amount = %v, want %s", out.PreviousAmount, previous)
	}
	if out.Achieved != nil {
		t.Errorf("achieved = %v, want nil (mês em curso)", out.Achieved)
	}
}

func TestGoalCreate_MissingRequiredField(t *testing.T) {
	repo := &mockReductionGoalRepository{
		CreateFn: func(context.Context, *entity.ReductionGoal) error {
			t.Fatal("repo.Create should not be called on validation failure")
			return nil
		},
	}
	uc := NewCreateUseCase(repo, &mockTransactionClient{}, testLogger())

	in := validCreateInput()
	in.CategoryID = uuid.Nil
	_, err := uc.Execute(context.Background(), in)
	if !errors.Is(err, ErrMissingRequiredField) {
		t.Fatalf("err = %v, want ErrMissingRequiredField", err)
	}
}

func TestGoalCreate_InvalidTargetAmount(t *testing.T) {
	uc := NewCreateUseCase(&mockReductionGoalRepository{}, &mockTransactionClient{}, testLogger())

	in := validCreateInput()
	in.TargetAmount = decimal.Zero
	_, err := uc.Execute(context.Background(), in)
	if !errors.Is(err, ErrInvalidTargetAmount) {
		t.Fatalf("err = %v, want ErrInvalidTargetAmount", err)
	}
}

func TestGoalCreate_InvalidCompetence(t *testing.T) {
	uc := NewCreateUseCase(&mockReductionGoalRepository{}, &mockTransactionClient{}, testLogger())

	in := validCreateInput()
	in.Competence = "2026/06"
	_, err := uc.Execute(context.Background(), in)
	if !errors.Is(err, ErrInvalidCompetence) {
		t.Fatalf("err = %v, want ErrInvalidCompetence", err)
	}
}

func TestGoalCreate_AlreadyExists(t *testing.T) {
	repo := &mockReductionGoalRepository{
		ExistsByUserCategoryAndCompetenceFn: func(context.Context, uuid.UUID, uuid.UUID, string) (bool, error) {
			return true, nil
		},
		CreateFn: func(context.Context, *entity.ReductionGoal) error {
			t.Fatal("repo.Create should not be called when goal already exists")
			return nil
		},
	}
	uc := NewCreateUseCase(repo, &mockTransactionClient{}, testLogger())

	_, err := uc.Execute(context.Background(), validCreateInput())
	if !errors.Is(err, ErrGoalAlreadyExists) {
		t.Fatalf("err = %v, want ErrGoalAlreadyExists", err)
	}
}

// TestGoalCreate_PreviousAmountNull_WhenUpstreamError cobre a degradação graciosa
// do snapshot (FDD-003 §4): falha upstream não bloqueia a criação da meta.
func TestGoalCreate_PreviousAmountNull_WhenUpstreamError(t *testing.T) {
	repo := &mockReductionGoalRepository{
		ExistsByUserCategoryAndCompetenceFn: func(context.Context, uuid.UUID, uuid.UUID, string) (bool, error) {
			return false, nil
		},
		CreateFn: func(_ context.Context, goal *entity.ReductionGoal) error {
			goal.ID = uuid.New()
			return nil
		},
	}
	client := &mockTransactionClient{
		GetExpensesByCategoryFn: func(context.Context, string, string, string) (*httpclient.ExpensesByCategoryOutput, error) {
			return nil, httpclient.ErrUpstreamTimeout
		},
	}
	uc := NewCreateUseCase(repo, client, testLogger())

	out, err := uc.Execute(context.Background(), validCreateInput())
	if err != nil {
		t.Fatalf("unexpected error: %v (upstream failure must not block creation)", err)
	}
	if out.PreviousAmount != nil {
		t.Errorf("previous_amount = %v, want nil on upstream failure", out.PreviousAmount)
	}
}

// TestGoalCreate_ExistsCheckError cobre a propagação de um erro técnico do
// repositório na checagem de unicidade (distinto de ErrGoalAlreadyExists).
func TestGoalCreate_ExistsCheckError(t *testing.T) {
	dbErr := errors.New("db unavailable")
	repo := &mockReductionGoalRepository{
		ExistsByUserCategoryAndCompetenceFn: func(context.Context, uuid.UUID, uuid.UUID, string) (bool, error) {
			return false, dbErr
		},
		CreateFn: func(context.Context, *entity.ReductionGoal) error {
			t.Fatal("repo.Create should not be called when the exists check fails")
			return nil
		},
	}
	uc := NewCreateUseCase(repo, &mockTransactionClient{}, testLogger())

	_, err := uc.Execute(context.Background(), validCreateInput())
	if !errors.Is(err, dbErr) {
		t.Fatalf("err = %v, want dbErr propagated", err)
	}
}

// TestGoalCreate_RepositoryError cobre a propagação de um erro técnico do
// repositório na persistência (Create), após todas as validações passarem.
func TestGoalCreate_RepositoryError(t *testing.T) {
	dbErr := errors.New("db unavailable")
	repo := &mockReductionGoalRepository{
		ExistsByUserCategoryAndCompetenceFn: func(context.Context, uuid.UUID, uuid.UUID, string) (bool, error) {
			return false, nil
		},
		CreateFn: func(context.Context, *entity.ReductionGoal) error {
			return dbErr
		},
	}
	client := &mockTransactionClient{
		GetExpensesByCategoryFn: func(context.Context, string, string, string) (*httpclient.ExpensesByCategoryOutput, error) {
			return nil, httpclient.ErrUpstreamTimeout
		},
	}
	uc := NewCreateUseCase(repo, client, testLogger())

	_, err := uc.Execute(context.Background(), validCreateInput())
	if !errors.Is(err, dbErr) {
		t.Fatalf("err = %v, want dbErr propagated", err)
	}
}
