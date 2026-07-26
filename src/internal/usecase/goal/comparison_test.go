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

func comparisonGoal(userID, categoryID uuid.UUID, target string, previous *decimal.Decimal) entity.ReductionGoal {
	return entity.ReductionGoal{
		ID:             uuid.New(),
		UserID:         userID,
		CategoryID:     categoryID,
		Competence:     "2026-06",
		TargetAmount:   decimal.RequireFromString(target),
		PreviousAmount: previous,
	}
}

func TestGoalComparison_Success(t *testing.T) {
	userID := uuid.New()
	categoryID := uuid.New()
	previous := decimal.RequireFromString("800.00")
	goalEnt := comparisonGoal(userID, categoryID, "700.00", &previous)
	repo := &mockReductionGoalRepository{
		ListByUserAndCompetenceFn: func(context.Context, uuid.UUID, string) ([]entity.ReductionGoal, error) {
			return []entity.ReductionGoal{goalEnt}, nil
		},
	}
	client := &mockTransactionClient{
		GetExpensesByCategoryFn: func(context.Context, string, string, string) (*httpclient.ExpensesByCategoryOutput, error) {
			return &httpclient.ExpensesByCategoryOutput{CategoryID: categoryID.String(), CategoryName: "Alimentação", Total: decimal.RequireFromString("620.00")}, nil
		},
	}
	uc := NewComparisonUseCase(repo, client, testLogger())

	items, err := uc.Execute(context.Background(), ComparisonInput{UserID: userID, Competence: "2026-06"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	it := items[0]
	if it.CategoryID != categoryID {
		t.Errorf("category_id = %s, want %s", it.CategoryID, categoryID)
	}
	if it.CategoryName == nil || *it.CategoryName != "Alimentação" {
		t.Errorf("category_name = %v, want Alimentação (CAL-05)", it.CategoryName)
	}
	if it.CurrentMonthAmount == nil || !it.CurrentMonthAmount.Equal(decimal.RequireFromString("620.00")) {
		t.Errorf("current_month_amount = %v, want 620.00", it.CurrentMonthAmount)
	}
	if it.OnTrack == nil || !*it.OnTrack {
		t.Errorf("on_track = %v, want true", it.OnTrack)
	}
	if it.VariationPct == nil || *it.VariationPct != -22.5 {
		t.Errorf("variation_pct = %v, want -22.5", it.VariationPct)
	}
	if it.VariationLabel == nil || *it.VariationLabel != "22,5% menor que o mês passado" {
		t.Errorf("variation_label = %v, want '22,5%% menor que o mês passado'", it.VariationLabel)
	}
	if it.TargetProgressPct == nil || *it.TargetProgressPct != 88.6 {
		t.Errorf("target_progress_pct = %v, want 88.6", it.TargetProgressPct)
	}
}

func TestGoalComparison_MissingRequiredField(t *testing.T) {
	repo := &mockReductionGoalRepository{
		ListByUserAndCompetenceFn: func(context.Context, uuid.UUID, string) ([]entity.ReductionGoal, error) {
			t.Fatal("repo should not be called on validation failure")
			return nil, nil
		},
	}
	uc := NewComparisonUseCase(repo, &mockTransactionClient{}, testLogger())

	_, err := uc.Execute(context.Background(), ComparisonInput{UserID: uuid.Nil, Competence: "2026-06"})
	if !errors.Is(err, ErrMissingRequiredField) {
		t.Fatalf("err = %v, want ErrMissingRequiredField", err)
	}
}

func TestGoalComparison_InvalidCompetence(t *testing.T) {
	uc := NewComparisonUseCase(&mockReductionGoalRepository{}, &mockTransactionClient{}, testLogger())

	_, err := uc.Execute(context.Background(), ComparisonInput{UserID: uuid.New(), Competence: "2026-13"})
	if !errors.Is(err, ErrInvalidCompetence) {
		t.Fatalf("err = %v, want ErrInvalidCompetence", err)
	}
}

func TestGoalComparison_NoPreviousAmount_NullVariation(t *testing.T) {
	userID := uuid.New()
	categoryID := uuid.New()
	goalEnt := comparisonGoal(userID, categoryID, "700.00", nil)
	repo := &mockReductionGoalRepository{
		ListByUserAndCompetenceFn: func(context.Context, uuid.UUID, string) ([]entity.ReductionGoal, error) {
			return []entity.ReductionGoal{goalEnt}, nil
		},
		UpdatePreviousAmountFn: func(context.Context, uuid.UUID, decimal.Decimal) error {
			t.Error("UpdatePreviousAmount should not be called when backfill fails")
			return nil
		},
	}
	client := &mockTransactionClient{
		GetExpensesByCategoryFn: func(_ context.Context, _, _, competence string) (*httpclient.ExpensesByCategoryOutput, error) {
			if competence == "2026-06" {
				return &httpclient.ExpensesByCategoryOutput{Total: decimal.RequireFromString("620.00")}, nil
			}
			return nil, errors.New("upstream unavailable")
		},
	}
	uc := NewComparisonUseCase(repo, client, testLogger())

	items, err := uc.Execute(context.Background(), ComparisonInput{UserID: userID, Competence: "2026-06"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	it := items[0]
	if it.PreviousMonthAmount != nil {
		t.Errorf("previous_month_amount = %v, want nil", it.PreviousMonthAmount)
	}
	if it.VariationPct != nil || it.VariationLabel != nil {
		t.Errorf("variation = (%v, %v), want nils", it.VariationPct, it.VariationLabel)
	}
	if it.OnTrack == nil || !*it.OnTrack {
		t.Errorf("on_track = %v, want true (independe da variação)", it.OnTrack)
	}
}

func TestGoalComparison_BackfillPersistsPreviousAmount(t *testing.T) {
	userID := uuid.New()
	categoryID := uuid.New()
	goalEnt := comparisonGoal(userID, categoryID, "700.00", nil)
	var persisted *decimal.Decimal
	repo := &mockReductionGoalRepository{
		ListByUserAndCompetenceFn: func(context.Context, uuid.UUID, string) ([]entity.ReductionGoal, error) {
			return []entity.ReductionGoal{goalEnt}, nil
		},
		UpdatePreviousAmountFn: func(_ context.Context, id uuid.UUID, amount decimal.Decimal) error {
			if id != goalEnt.ID {
				t.Errorf("UpdatePreviousAmount id = %s, want %s", id, goalEnt.ID)
			}
			persisted = &amount
			return nil
		},
	}
	client := &mockTransactionClient{
		GetExpensesByCategoryFn: func(_ context.Context, _, _, competence string) (*httpclient.ExpensesByCategoryOutput, error) {
			switch competence {
			case "2026-06":
				return &httpclient.ExpensesByCategoryOutput{Total: decimal.RequireFromString("620.00")}, nil
			case "2026-05":
				return &httpclient.ExpensesByCategoryOutput{Total: decimal.RequireFromString("800.00")}, nil
			}
			return nil, errors.New("unexpected competence")
		},
	}
	uc := NewComparisonUseCase(repo, client, testLogger())

	items, err := uc.Execute(context.Background(), ComparisonInput{UserID: userID, Competence: "2026-06"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if persisted == nil || !persisted.Equal(decimal.RequireFromString("800.00")) {
		t.Errorf("persisted previous_amount = %v, want 800.00", persisted)
	}
	it := items[0]
	if it.VariationPct == nil || *it.VariationPct != -22.5 {
		t.Errorf("variation_pct = %v, want -22.5 (calculada sobre o backfill)", it.VariationPct)
	}
}

func TestGoalComparison_UpstreamUnavailable_PartialNulls(t *testing.T) {
	userID := uuid.New()
	okCategory := uuid.New()
	failCategory := uuid.New()
	prevOK := decimal.RequireFromString("500.00")
	prevFail := decimal.RequireFromString("300.00")
	goals := []entity.ReductionGoal{
		comparisonGoal(userID, okCategory, "700.00", &prevOK),
		comparisonGoal(userID, failCategory, "200.00", &prevFail),
	}
	repo := &mockReductionGoalRepository{
		ListByUserAndCompetenceFn: func(context.Context, uuid.UUID, string) ([]entity.ReductionGoal, error) {
			return goals, nil
		},
	}
	client := &mockTransactionClient{
		GetExpensesByCategoryFn: func(_ context.Context, _, categoryID, _ string) (*httpclient.ExpensesByCategoryOutput, error) {
			if categoryID == failCategory.String() {
				return nil, errors.New("upstream unavailable")
			}
			return &httpclient.ExpensesByCategoryOutput{Total: decimal.RequireFromString("400.00")}, nil
		},
	}
	uc := NewComparisonUseCase(repo, client, testLogger())

	items, err := uc.Execute(context.Background(), ComparisonInput{UserID: userID, Competence: "2026-06"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2 (item degradado deve permanecer na lista)", len(items))
	}
	ok, degraded := items[0], items[1]
	if ok.CurrentMonthAmount == nil || ok.OnTrack == nil {
		t.Errorf("item saudável degradado indevidamente: %+v", ok)
	}
	if degraded.CurrentMonthAmount != nil || degraded.OnTrack != nil ||
		degraded.VariationPct != nil || degraded.TargetProgressPct != nil {
		t.Errorf("item com upstream fora deveria ter campos nulos: %+v", degraded)
	}
	if degraded.PreviousMonthAmount == nil || !degraded.PreviousMonthAmount.Equal(prevFail) {
		t.Errorf("previous_month_amount deve ser mantido do snapshot: %v", degraded.PreviousMonthAmount)
	}
}

func TestGoalComparison_RepositoryError(t *testing.T) {
	repoErr := errors.New("db down")
	repo := &mockReductionGoalRepository{
		ListByUserAndCompetenceFn: func(context.Context, uuid.UUID, string) ([]entity.ReductionGoal, error) {
			return nil, repoErr
		},
	}
	uc := NewComparisonUseCase(repo, &mockTransactionClient{}, testLogger())

	_, err := uc.Execute(context.Background(), ComparisonInput{UserID: uuid.New(), Competence: "2026-06"})
	if !errors.Is(err, repoErr) {
		t.Fatalf("err = %v, want repository error propagated", err)
	}
}
