package global_budget

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
	"github.com/victorgmrqs/hf-income-service/src/pkg/httpclient"
)

// autoAdjustFixture monta os mocks do cenário feliz: teto anterior 5000.00 em
// 2026-06 e upsert que apenas gera o ID. Os testes sobrescrevem o necessário.
func autoAdjustFixture(prevCeiling string) (*mockGlobalBudgetRepository, *mockTransactionClient) {
	repo := &mockGlobalBudgetRepository{
		GetByUserAndCompetenceFn: func(_ context.Context, userID uuid.UUID, competence string) (*entity.GlobalBudget, error) {
			return &entity.GlobalBudget{
				ID:         uuid.New(),
				UserID:     userID,
				Competence: competence,
				Ceiling:    decimal.RequireFromString(prevCeiling),
			}, nil
		},
		UpsertFn: func(_ context.Context, budget *entity.GlobalBudget) error {
			budget.ID = uuid.New()
			return nil
		},
	}
	client := &mockTransactionClient{}
	return repo, client
}

func totalsWith(general string) func(context.Context, string, string) (*httpclient.ExpenseTotalsOutput, error) {
	return func(_ context.Context, _, competence string) (*httpclient.ExpenseTotalsOutput, error) {
		return &httpclient.ExpenseTotalsOutput{
			Competence:   competence,
			TotalGeneral: decimal.RequireFromString(general),
		}, nil
	}
}

func TestAutoAdjustGlobalBudget_SpendingBelowCeiling(t *testing.T) {
	repo, client := autoAdjustFixture("5000.00")
	client.GetExpenseTotalsFn = totalsWith("4200.00")

	var prevCompetenceAsked, totalsCompetenceAsked string
	baseGet := repo.GetByUserAndCompetenceFn
	repo.GetByUserAndCompetenceFn = func(ctx context.Context, userID uuid.UUID, competence string) (*entity.GlobalBudget, error) {
		prevCompetenceAsked = competence
		return baseGet(ctx, userID, competence)
	}
	baseTotals := client.GetExpenseTotalsFn
	client.GetExpenseTotalsFn = func(ctx context.Context, userID, competence string) (*httpclient.ExpenseTotalsOutput, error) {
		totalsCompetenceAsked = competence
		return baseTotals(ctx, userID, competence)
	}

	uc := NewAutoAdjustUseCase(repo, client, testLogger())
	out, err := uc.Execute(context.Background(), AutoAdjustInput{UserID: uuid.New(), Competence: "2026-07"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ORC-03: gasto < teto → novo teto = gasto, auto_adjusted = true.
	if !out.Ceiling.Equal(decimal.RequireFromString("4200.00")) {
		t.Errorf("ceiling = %s, want 4200.00", out.Ceiling)
	}
	if !out.AutoAdjusted {
		t.Error("auto_adjusted = false, want true (ORC-03)")
	}
	if prevCompetenceAsked != "2026-06" {
		t.Errorf("previous budget competence = %q, want 2026-06", prevCompetenceAsked)
	}
	if totalsCompetenceAsked != "2026-06" {
		t.Errorf("expense totals competence = %q, want 2026-06", totalsCompetenceAsked)
	}
	if out.Competence != "2026-07" {
		t.Errorf("competence = %q, want 2026-07", out.Competence)
	}
}

func TestAutoAdjustGlobalBudget_SpendingAtOrAboveCeiling(t *testing.T) {
	for _, spending := range []string{"5000.00", "6100.00"} {
		repo, client := autoAdjustFixture("5000.00")
		client.GetExpenseTotalsFn = totalsWith(spending)

		uc := NewAutoAdjustUseCase(repo, client, testLogger())
		out, err := uc.Execute(context.Background(), AutoAdjustInput{UserID: uuid.New(), Competence: "2026-07"})
		if err != nil {
			t.Fatalf("spending %s: unexpected error: %v", spending, err)
		}
		// ORC-04: gasto >= teto → teto mantido, auto_adjusted = false.
		if !out.Ceiling.Equal(decimal.RequireFromString("5000.00")) {
			t.Errorf("spending %s: ceiling = %s, want 5000.00", spending, out.Ceiling)
		}
		if out.AutoAdjusted {
			t.Errorf("spending %s: auto_adjusted = true, want false (ORC-04)", spending)
		}
	}
}

func TestAutoAdjustGlobalBudget_NoPreviousBudget(t *testing.T) {
	repo, client := autoAdjustFixture("5000.00")
	repo.GetByUserAndCompetenceFn = func(context.Context, uuid.UUID, string) (*entity.GlobalBudget, error) {
		return nil, gorm.ErrRecordNotFound
	}
	repo.UpsertFn = func(context.Context, *entity.GlobalBudget) error {
		t.Fatal("repo.Upsert should not be called without a previous budget")
		return nil
	}

	uc := NewAutoAdjustUseCase(repo, client, testLogger())
	_, err := uc.Execute(context.Background(), AutoAdjustInput{UserID: uuid.New(), Competence: "2026-07"})
	if !errors.Is(err, ErrNoPreviousBudget) {
		t.Fatalf("err = %v, want ErrNoPreviousBudget", err)
	}
}

func TestAutoAdjustGlobalBudget_Idempotent(t *testing.T) {
	// Upsert em memória: a segunda execução atualiza o mesmo registro em vez de duplicar.
	store := map[string]*entity.GlobalBudget{}
	repo, client := autoAdjustFixture("5000.00")
	client.GetExpenseTotalsFn = totalsWith("4200.00")
	repo.UpsertFn = func(_ context.Context, budget *entity.GlobalBudget) error {
		key := budget.UserID.String() + budget.Competence
		if existing, ok := store[key]; ok {
			existing.Ceiling = budget.Ceiling
			existing.AutoAdjusted = budget.AutoAdjusted
			*budget = *existing
			return nil
		}
		budget.ID = uuid.New()
		clone := *budget
		store[key] = &clone
		return nil
	}

	uc := NewAutoAdjustUseCase(repo, client, testLogger())
	in := AutoAdjustInput{UserID: uuid.New(), Competence: "2026-07"}

	first, err := uc.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	second, err := uc.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if first.ID != second.ID {
		t.Errorf("second run created a new record: id %s != %s (want upsert)", second.ID, first.ID)
	}
	if !second.Ceiling.Equal(first.Ceiling) {
		t.Errorf("second run ceiling = %s, want %s", second.Ceiling, first.Ceiling)
	}
	if len(store) != 1 {
		t.Errorf("store has %d records, want 1", len(store))
	}
}

func TestAutoAdjustGlobalBudget_UpstreamError(t *testing.T) {
	repo, client := autoAdjustFixture("5000.00")
	client.GetExpenseTotalsFn = func(context.Context, string, string) (*httpclient.ExpenseTotalsOutput, error) {
		return nil, fmt.Errorf("hf-transaction-service returned status 503")
	}
	repo.UpsertFn = func(context.Context, *entity.GlobalBudget) error {
		t.Fatal("repo.Upsert should not be called on upstream failure")
		return nil
	}

	uc := NewAutoAdjustUseCase(repo, client, testLogger())
	_, err := uc.Execute(context.Background(), AutoAdjustInput{UserID: uuid.New(), Competence: "2026-07"})
	if !errors.Is(err, ErrUpstreamUnavailable) {
		t.Fatalf("err = %v, want ErrUpstreamUnavailable", err)
	}
}

func TestAutoAdjustGlobalBudget_MissingRequiredField(t *testing.T) {
	repo, client := autoAdjustFixture("5000.00")
	uc := NewAutoAdjustUseCase(repo, client, testLogger())

	_, err := uc.Execute(context.Background(), AutoAdjustInput{UserID: uuid.New(), Competence: ""})
	if !errors.Is(err, ErrMissingRequiredField) {
		t.Fatalf("err = %v, want ErrMissingRequiredField", err)
	}
}

func TestAutoAdjustGlobalBudget_InvalidCompetence(t *testing.T) {
	repo, client := autoAdjustFixture("5000.00")
	uc := NewAutoAdjustUseCase(repo, client, testLogger())

	_, err := uc.Execute(context.Background(), AutoAdjustInput{UserID: uuid.New(), Competence: "2026/07"})
	if !errors.Is(err, ErrInvalidCompetence) {
		t.Fatalf("err = %v, want ErrInvalidCompetence", err)
	}
}

func TestAutoAdjustGlobalBudget_ZeroSpending(t *testing.T) {
	// FDD-002 §4 (fluxo alternativo): sem gastos no mês anterior → teto 0 + WARN.
	repo, client := autoAdjustFixture("5000.00")
	client.GetExpenseTotalsFn = totalsWith("0")

	uc := NewAutoAdjustUseCase(repo, client, testLogger())
	out, err := uc.Execute(context.Background(), AutoAdjustInput{UserID: uuid.New(), Competence: "2026-07"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Ceiling.IsZero() {
		t.Errorf("ceiling = %s, want 0", out.Ceiling)
	}
	if !out.AutoAdjusted {
		t.Error("auto_adjusted = false, want true (0 < teto anterior)")
	}
}
