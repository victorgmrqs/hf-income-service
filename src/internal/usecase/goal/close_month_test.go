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

func TestGoalCloseMonth_Success_SetsAchievedTrueAndFalse(t *testing.T) {
	userID := uuid.New()
	underTarget := comparisonGoal(userID, uuid.New(), "700.00", nil) // gasto 620 ≤ 700 → true
	overTarget := comparisonGoal(userID, uuid.New(), "300.00", nil)  // gasto 350 > 300 → false
	achievedByID := map[uuid.UUID]bool{}
	repo := &mockReductionGoalRepository{
		ListWithNullAchievedByCompetenceFn: func(context.Context, string) ([]entity.ReductionGoal, error) {
			return []entity.ReductionGoal{underTarget, overTarget}, nil
		},
		SetAchievedFn: func(_ context.Context, id uuid.UUID, achieved bool) error {
			achievedByID[id] = achieved
			return nil
		},
	}
	client := &mockTransactionClient{
		GetExpensesByCategoryFn: func(_ context.Context, _, categoryID, _ string) (*httpclient.ExpensesByCategoryOutput, error) {
			if categoryID == underTarget.CategoryID.String() {
				return &httpclient.ExpensesByCategoryOutput{Total: decimal.RequireFromString("620.00")}, nil
			}
			return &httpclient.ExpensesByCategoryOutput{Total: decimal.RequireFromString("350.00")}, nil
		},
	}
	uc := NewCloseMonthUseCase(repo, client, testLogger())

	out, err := uc.Execute(context.Background(), CloseMonthInput{Competence: "2026-06"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Closed != 2 {
		t.Errorf("closed = %d, want 2", out.Closed)
	}
	if achieved, ok := achievedByID[underTarget.ID]; !ok || !achieved {
		t.Errorf("meta abaixo do teto: achieved = %v, want true (MET-06)", achieved)
	}
	if achieved, ok := achievedByID[overTarget.ID]; !ok || achieved {
		t.Errorf("meta acima do teto: achieved = %v, want false (MET-06)", achieved)
	}
}

func TestGoalCloseMonth_NoOpenGoals_ClosedZero(t *testing.T) {
	repo := &mockReductionGoalRepository{
		ListWithNullAchievedByCompetenceFn: func(context.Context, string) ([]entity.ReductionGoal, error) {
			return nil, nil
		},
	}
	uc := NewCloseMonthUseCase(repo, &mockTransactionClient{}, testLogger())

	out, err := uc.Execute(context.Background(), CloseMonthInput{Competence: "2026-06"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Closed != 0 {
		t.Errorf("closed = %d, want 0", out.Closed)
	}
}

func TestGoalCloseMonth_MissingRequiredField(t *testing.T) {
	repo := &mockReductionGoalRepository{
		ListWithNullAchievedByCompetenceFn: func(context.Context, string) ([]entity.ReductionGoal, error) {
			t.Fatal("repo should not be called on validation failure")
			return nil, nil
		},
	}
	uc := NewCloseMonthUseCase(repo, &mockTransactionClient{}, testLogger())

	_, err := uc.Execute(context.Background(), CloseMonthInput{Competence: ""})
	if !errors.Is(err, ErrMissingRequiredField) {
		t.Fatalf("err = %v, want ErrMissingRequiredField", err)
	}
}

func TestGoalCloseMonth_InvalidCompetence(t *testing.T) {
	uc := NewCloseMonthUseCase(&mockReductionGoalRepository{}, &mockTransactionClient{}, testLogger())

	_, err := uc.Execute(context.Background(), CloseMonthInput{Competence: "06-2026"})
	if !errors.Is(err, ErrInvalidCompetence) {
		t.Fatalf("err = %v, want ErrInvalidCompetence", err)
	}
}

func TestGoalCloseMonth_UpstreamUnavailable_SkipsGoal(t *testing.T) {
	userID := uuid.New()
	healthy := comparisonGoal(userID, uuid.New(), "700.00", nil)
	broken := comparisonGoal(userID, uuid.New(), "300.00", nil)
	setCalls := 0
	repo := &mockReductionGoalRepository{
		ListWithNullAchievedByCompetenceFn: func(context.Context, string) ([]entity.ReductionGoal, error) {
			return []entity.ReductionGoal{healthy, broken}, nil
		},
		SetAchievedFn: func(_ context.Context, id uuid.UUID, _ bool) error {
			if id == broken.ID {
				t.Error("SetAchieved should not be called for the skipped goal")
			}
			setCalls++
			return nil
		},
	}
	client := &mockTransactionClient{
		GetExpensesByCategoryFn: func(_ context.Context, _, categoryID, _ string) (*httpclient.ExpensesByCategoryOutput, error) {
			if categoryID == broken.CategoryID.String() {
				return nil, errors.New("upstream unavailable")
			}
			return &httpclient.ExpensesByCategoryOutput{Total: decimal.RequireFromString("100.00")}, nil
		},
	}
	uc := NewCloseMonthUseCase(repo, client, testLogger())

	out, err := uc.Execute(context.Background(), CloseMonthInput{Competence: "2026-06"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Closed != 1 || setCalls != 1 {
		t.Errorf("closed = %d (setCalls = %d), want 1 — meta com upstream fora é pulada e as demais seguem", out.Closed, setCalls)
	}
}

func TestGoalCloseMonth_PersistError_SkipsGoal(t *testing.T) {
	goalEnt := comparisonGoal(uuid.New(), uuid.New(), "700.00", nil)
	repo := &mockReductionGoalRepository{
		ListWithNullAchievedByCompetenceFn: func(context.Context, string) ([]entity.ReductionGoal, error) {
			return []entity.ReductionGoal{goalEnt}, nil
		},
		SetAchievedFn: func(context.Context, uuid.UUID, bool) error {
			return errors.New("db down")
		},
	}
	client := &mockTransactionClient{
		GetExpensesByCategoryFn: func(context.Context, string, string, string) (*httpclient.ExpensesByCategoryOutput, error) {
			return &httpclient.ExpensesByCategoryOutput{Total: decimal.RequireFromString("100.00")}, nil
		},
	}
	uc := NewCloseMonthUseCase(repo, client, testLogger())

	out, err := uc.Execute(context.Background(), CloseMonthInput{Competence: "2026-06"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Closed != 0 {
		t.Errorf("closed = %d, want 0 (falha de persistência não conta como fechada)", out.Closed)
	}
}

func TestGoalCloseMonth_RepositoryError(t *testing.T) {
	repoErr := errors.New("db down")
	repo := &mockReductionGoalRepository{
		ListWithNullAchievedByCompetenceFn: func(context.Context, string) ([]entity.ReductionGoal, error) {
			return nil, repoErr
		},
	}
	uc := NewCloseMonthUseCase(repo, &mockTransactionClient{}, testLogger())

	_, err := uc.Execute(context.Background(), CloseMonthInput{Competence: "2026-06"})
	if !errors.Is(err, repoErr) {
		t.Fatalf("err = %v, want repository error propagated", err)
	}
}
