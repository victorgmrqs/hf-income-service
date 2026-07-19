package balance

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

func validInput() GetInput {
	return GetInput{UserID: uuid.New(), Competence: "2026-06"}
}

func TestGetBalance_Success_WithCeiling(t *testing.T) {
	incomeRepo, budgetRepo, client := happyFixture()

	var dueDateAsked string
	baseAP := client.GetAccountsPayableFn
	client.GetAccountsPayableFn = func(ctx context.Context, userID, dueDateUntil string) ([]httpclient.PendingBillOutput, error) {
		dueDateAsked = dueDateUntil
		return baseAP(ctx, userID, dueDateUntil)
	}

	uc := NewGetUseCase(incomeRepo, budgetRepo, client, testLogger())
	out, err := uc.Execute(context.Background(), validInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// FDD-004 §5: cenário de referência completo.
	assertDecimal(t, "total_income", out.TotalIncome, "7500.00")
	assertDecimal(t, "total_personal", out.TotalPersonal, "1800.00")
	assertDecimal(t, "total_shared", out.TotalShared, "1400.00")
	assertDecimal(t, "total_expenses", out.TotalExpenses, "3200.00")
	assertDecimal(t, "balance_today", out.BalanceToday, "4300.00")         // SAL-01
	assertDecimal(t, "committed_bills", out.CommittedBills, "850.00")      // SAL-02
	assertDecimal(t, "projected_balance", out.ProjectedBalance, "3450.00") // SAL-03
	if out.IsProjectedNegative {
		t.Error("is_projected_negative = true, want false")
	}
	if out.Ceiling == nil || !out.Ceiling.Equal(decimal.RequireFromString("5000.00")) {
		t.Errorf("ceiling = %v, want 5000.00", out.Ceiling)
	}
	if out.CeilingUsagePct == nil || *out.CeilingUsagePct != 64 {
		t.Errorf("ceiling_usage_pct = %v, want 64 (ORC-07, inteiro)", out.CeilingUsagePct)
	}
	if out.CeilingExceeded {
		t.Error("ceiling_exceeded = true, want false")
	}
	if dueDateAsked != "2026-06-30" {
		t.Errorf("due_date_until = %q, want 2026-06-30 (lastDayOfMonth)", dueDateAsked)
	}
}

func TestGetBalance_Success_WithoutCeiling(t *testing.T) {
	incomeRepo, budgetRepo, client := happyFixture()
	budgetRepo.GetByUserAndCompetenceFn = func(context.Context, uuid.UUID, string) (*entity.GlobalBudget, error) {
		return nil, gorm.ErrRecordNotFound
	}

	uc := NewGetUseCase(incomeRepo, budgetRepo, client, testLogger())
	out, err := uc.Execute(context.Background(), validInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ORC-06/07: sem teto → null/null/false; saldo calculado normalmente.
	if out.Ceiling != nil || out.CeilingUsagePct != nil {
		t.Errorf("ceiling = %v, usage = %v; want null/null", out.Ceiling, out.CeilingUsagePct)
	}
	if out.CeilingExceeded {
		t.Error("ceiling_exceeded = true, want false sem teto")
	}
	assertDecimal(t, "balance_today", out.BalanceToday, "4300.00")
}

func TestGetBalance_TotalIncomeZero_Returns200(t *testing.T) {
	incomeRepo, budgetRepo, client := happyFixture()
	incomeRepo.SumFn = func(context.Context, uuid.UUID, string) (decimal.Decimal, error) {
		return decimal.Zero, nil
	}

	uc := NewGetUseCase(incomeRepo, budgetRepo, client, testLogger())
	out, err := uc.Execute(context.Background(), validInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// SAL-04: campos sempre presentes; saldo negativo é resultado válido.
	assertDecimal(t, "balance_today", out.BalanceToday, "-3200.00")
	assertDecimal(t, "projected_balance", out.ProjectedBalance, "-4050.00")
	if !out.IsProjectedNegative {
		t.Error("is_projected_negative = false, want true")
	}
}

func TestGetBalance_IsProjectedNegative(t *testing.T) {
	incomeRepo, budgetRepo, client := happyFixture()
	// Contas pendentes maiores que o saldo do dia → projeção negativa (SAL-05).
	client.GetAccountsPayableFn = func(context.Context, string, string) ([]httpclient.PendingBillOutput, error) {
		return []httpclient.PendingBillOutput{
			{Amount: decimal.RequireFromString("4000.00"), Status: "PENDING"},
			{Amount: decimal.RequireFromString("500.00"), Status: "PENDING"},
		}, nil
	}

	uc := NewGetUseCase(incomeRepo, budgetRepo, client, testLogger())
	out, err := uc.Execute(context.Background(), validInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertDecimal(t, "committed_bills", out.CommittedBills, "4500.00")
	assertDecimal(t, "projected_balance", out.ProjectedBalance, "-200.00")
	if !out.IsProjectedNegative {
		t.Error("is_projected_negative = false, want true (SAL-05)")
	}
}

func TestGetBalance_CeilingExceeded(t *testing.T) {
	incomeRepo, budgetRepo, client := happyFixture()
	budgetRepo.GetByUserAndCompetenceFn = func(_ context.Context, userID uuid.UUID, competence string) (*entity.GlobalBudget, error) {
		return &entity.GlobalBudget{ID: uuid.New(), UserID: userID, Competence: competence,
			Ceiling: decimal.RequireFromString("2500.00")}, nil
	}

	uc := NewGetUseCase(incomeRepo, budgetRepo, client, testLogger())
	out, err := uc.Execute(context.Background(), validInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ORC-06: gasto 3200 > teto 2500 → exceeded; ORC-07: pct 128 (>100 = estouro).
	if !out.CeilingExceeded {
		t.Error("ceiling_exceeded = false, want true (ORC-06)")
	}
	if out.CeilingUsagePct == nil || *out.CeilingUsagePct != 128 {
		t.Errorf("ceiling_usage_pct = %v, want 128", out.CeilingUsagePct)
	}
}

func TestGetBalance_ZeroCeiling_UsagePctNull(t *testing.T) {
	// Teto 0 existe via auto-ajuste com gasto zero (FDD-002 §4): pct indefinido → null.
	incomeRepo, budgetRepo, client := happyFixture()
	budgetRepo.GetByUserAndCompetenceFn = func(_ context.Context, userID uuid.UUID, competence string) (*entity.GlobalBudget, error) {
		return &entity.GlobalBudget{ID: uuid.New(), UserID: userID, Competence: competence,
			Ceiling: decimal.Zero}, nil
	}

	uc := NewGetUseCase(incomeRepo, budgetRepo, client, testLogger())
	out, err := uc.Execute(context.Background(), validInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.CeilingUsagePct != nil {
		t.Errorf("ceiling_usage_pct = %v, want null com teto 0", out.CeilingUsagePct)
	}
	if !out.CeilingExceeded {
		t.Error("ceiling_exceeded = false, want true (gasto 3200 > teto 0)")
	}
}

func TestGetBalance_MissingRequiredField(t *testing.T) {
	incomeRepo, budgetRepo, client := happyFixture()
	uc := NewGetUseCase(incomeRepo, budgetRepo, client, testLogger())

	_, err := uc.Execute(context.Background(), GetInput{UserID: uuid.Nil, Competence: "2026-06"})
	if !errors.Is(err, ErrMissingRequiredField) {
		t.Fatalf("err = %v, want ErrMissingRequiredField", err)
	}
}

func TestGetBalance_InvalidCompetence(t *testing.T) {
	incomeRepo, budgetRepo, client := happyFixture()
	uc := NewGetUseCase(incomeRepo, budgetRepo, client, testLogger())

	_, err := uc.Execute(context.Background(), GetInput{UserID: uuid.New(), Competence: "2026/06"})
	if !errors.Is(err, ErrInvalidCompetence) {
		t.Fatalf("err = %v, want ErrInvalidCompetence", err)
	}
}

func TestGetBalance_UpstreamTimeout(t *testing.T) {
	incomeRepo, budgetRepo, client := happyFixture()
	client.GetExpenseTotalsFn = func(context.Context, string, string) (*httpclient.ExpenseTotalsOutput, error) {
		return nil, httpclient.ErrUpstreamTimeout
	}

	uc := NewGetUseCase(incomeRepo, budgetRepo, client, testLogger())
	_, err := uc.Execute(context.Background(), validInput())
	if !errors.Is(err, ErrUpstreamTimeout) {
		t.Fatalf("err = %v, want ErrUpstreamTimeout (503)", err)
	}
}

func TestGetBalance_UpstreamError(t *testing.T) {
	incomeRepo, budgetRepo, client := happyFixture()
	client.GetExpenseTotalsFn = func(context.Context, string, string) (*httpclient.ExpenseTotalsOutput, error) {
		return nil, fmt.Errorf("%w: status 500", httpclient.ErrUpstreamError)
	}

	uc := NewGetUseCase(incomeRepo, budgetRepo, client, testLogger())
	_, err := uc.Execute(context.Background(), validInput())
	if !errors.Is(err, ErrUpstreamError) {
		t.Fatalf("err = %v, want ErrUpstreamError (502)", err)
	}
}

func TestGetBalance_UpstreamError_AccountsPayable(t *testing.T) {
	incomeRepo, budgetRepo, client := happyFixture()
	client.GetAccountsPayableFn = func(context.Context, string, string) ([]httpclient.PendingBillOutput, error) {
		return nil, fmt.Errorf("%w: status 502", httpclient.ErrUpstreamError)
	}

	uc := NewGetUseCase(incomeRepo, budgetRepo, client, testLogger())
	out, err := uc.Execute(context.Background(), validInput())
	if !errors.Is(err, ErrUpstreamError) {
		t.Fatalf("err = %v, want ErrUpstreamError", err)
	}
	// Nunca dados parciais (FDD-004 invariante).
	if out != nil {
		t.Error("output != nil em falha upstream — dados parciais proibidos")
	}
}

func TestGetBalance_LocalRepoError(t *testing.T) {
	incomeRepo, budgetRepo, client := happyFixture()
	incomeRepo.SumFn = func(context.Context, uuid.UUID, string) (decimal.Decimal, error) {
		return decimal.Zero, errors.New("connection refused")
	}

	uc := NewGetUseCase(incomeRepo, budgetRepo, client, testLogger())
	_, err := uc.Execute(context.Background(), validInput())
	if err == nil {
		t.Fatal("expected error from local repo failure")
	}
	// Falha local não vira erro upstream (500, não 502/503).
	if errors.Is(err, ErrUpstreamTimeout) || errors.Is(err, ErrUpstreamError) {
		t.Fatalf("err = %v, não deveria mapear para erro upstream", err)
	}
}

func assertDecimal(t *testing.T, field string, got decimal.Decimal, want string) {
	t.Helper()
	if !got.Equal(decimal.RequireFromString(want)) {
		t.Errorf("%s = %s, want %s", field, got, want)
	}
}
