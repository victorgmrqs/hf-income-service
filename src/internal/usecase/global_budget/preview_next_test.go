package global_budget

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
	"github.com/victorgmrqs/hf-income-service/src/pkg/httpclient"
)

// newPreviewUseCase monta o use case com relógio fixo (competência corrente 2026-06).
func newPreviewUseCase(repo *mockGlobalBudgetRepository, client *mockTransactionClient) *previewNextUseCase {
	return &previewNextUseCase{
		repo:   repo,
		client: client,
		logger: testLogger(),
		nowFn:  func() time.Time { return time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC) },
	}
}

func TestPreviewNextGlobalBudget_Success(t *testing.T) {
	repo := &mockGlobalBudgetRepository{
		GetByUserAndCompetenceFn: func(_ context.Context, userID uuid.UUID, competence string) (*entity.GlobalBudget, error) {
			if competence != "2026-06" {
				t.Errorf("current competence = %q, want 2026-06", competence)
			}
			return &entity.GlobalBudget{
				ID:         uuid.New(),
				UserID:     userID,
				Competence: competence,
				Ceiling:    decimal.RequireFromString("5000.00"),
			}, nil
		},
	}
	client := &mockTransactionClient{GetExpenseTotalsFn: totalsWith("4200.00")}

	uc := newPreviewUseCase(repo, client)
	out, err := uc.Execute(context.Background(), PreviewInput{UserID: uuid.New()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.CurrentCompetence != "2026-06" || out.NextCompetence != "2026-07" {
		t.Errorf("competences = %s → %s, want 2026-06 → 2026-07", out.CurrentCompetence, out.NextCompetence)
	}
	if !out.SuggestedCeiling.Equal(decimal.RequireFromString("4200.00")) {
		t.Errorf("suggested_ceiling = %s, want 4200.00 (ORC-03)", out.SuggestedCeiling)
	}
	if out.AdjustmentReason != AdjustmentReasonSpendingBelowCeiling {
		t.Errorf("adjustment_reason = %q, want %q", out.AdjustmentReason, AdjustmentReasonSpendingBelowCeiling)
	}
	if !out.CurrentSpending.Equal(decimal.RequireFromString("4200.00")) {
		t.Errorf("current_spending = %s, want 4200.00", out.CurrentSpending)
	}
}

func TestPreviewNextGlobalBudget_SpendingAtCeiling_KeepsSuggestion(t *testing.T) {
	repo := &mockGlobalBudgetRepository{
		GetByUserAndCompetenceFn: func(_ context.Context, userID uuid.UUID, competence string) (*entity.GlobalBudget, error) {
			return &entity.GlobalBudget{ID: uuid.New(), UserID: userID, Competence: competence,
				Ceiling: decimal.RequireFromString("5000.00")}, nil
		},
	}
	client := &mockTransactionClient{GetExpenseTotalsFn: totalsWith("5000.00")}

	uc := newPreviewUseCase(repo, client)
	out, err := uc.Execute(context.Background(), PreviewInput{UserID: uuid.New()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ORC-04: gasto >= teto → sugestão mantém o teto.
	if !out.SuggestedCeiling.Equal(decimal.RequireFromString("5000.00")) {
		t.Errorf("suggested_ceiling = %s, want 5000.00 (ORC-04)", out.SuggestedCeiling)
	}
	if out.AdjustmentReason != AdjustmentReasonSpendingEqualsCeiling {
		t.Errorf("adjustment_reason = %q, want %q", out.AdjustmentReason, AdjustmentReasonSpendingEqualsCeiling)
	}
}

func TestPreviewNextGlobalBudget_NoPreviousBudget(t *testing.T) {
	repo := &mockGlobalBudgetRepository{
		GetByUserAndCompetenceFn: func(context.Context, uuid.UUID, string) (*entity.GlobalBudget, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}
	client := &mockTransactionClient{
		GetExpenseTotalsFn: func(context.Context, string, string) (*httpclient.ExpenseTotalsOutput, error) {
			t.Fatal("httpclient should not be called without a current budget")
			return nil, nil
		},
	}

	uc := newPreviewUseCase(repo, client)
	_, err := uc.Execute(context.Background(), PreviewInput{UserID: uuid.New()})
	if !errors.Is(err, ErrNoPreviousBudget) {
		t.Fatalf("err = %v, want ErrNoPreviousBudget", err)
	}
}

func TestPreviewNextGlobalBudget_UpstreamError(t *testing.T) {
	repo := &mockGlobalBudgetRepository{
		GetByUserAndCompetenceFn: func(_ context.Context, userID uuid.UUID, competence string) (*entity.GlobalBudget, error) {
			return &entity.GlobalBudget{ID: uuid.New(), UserID: userID, Competence: competence,
				Ceiling: decimal.RequireFromString("5000.00")}, nil
		},
	}
	client := &mockTransactionClient{
		GetExpenseTotalsFn: func(context.Context, string, string) (*httpclient.ExpenseTotalsOutput, error) {
			return nil, fmt.Errorf("call hf-transaction-service: connection refused")
		},
	}

	uc := newPreviewUseCase(repo, client)
	_, err := uc.Execute(context.Background(), PreviewInput{UserID: uuid.New()})
	if !errors.Is(err, ErrUpstreamUnavailable) {
		t.Fatalf("err = %v, want ErrUpstreamUnavailable", err)
	}
}

func TestPreviewNextGlobalBudget_YearRollover(t *testing.T) {
	repo := &mockGlobalBudgetRepository{
		GetByUserAndCompetenceFn: func(_ context.Context, userID uuid.UUID, competence string) (*entity.GlobalBudget, error) {
			return &entity.GlobalBudget{ID: uuid.New(), UserID: userID, Competence: competence,
				Ceiling: decimal.RequireFromString("5000.00")}, nil
		},
	}
	client := &mockTransactionClient{GetExpenseTotalsFn: totalsWith("4200.00")}

	uc := newPreviewUseCase(repo, client)
	uc.nowFn = func() time.Time { return time.Date(2026, 12, 20, 12, 0, 0, 0, time.UTC) }

	out, err := uc.Execute(context.Background(), PreviewInput{UserID: uuid.New()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.CurrentCompetence != "2026-12" || out.NextCompetence != "2027-01" {
		t.Errorf("competences = %s → %s, want 2026-12 → 2027-01", out.CurrentCompetence, out.NextCompetence)
	}
}
