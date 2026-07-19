package balance

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
	"github.com/victorgmrqs/hf-income-service/src/pkg/httpclient"
	"github.com/victorgmrqs/hf-income-service/src/pkg/observability"
)

func testLogger() *slog.Logger { return observability.NewLogger("test") }

// stubIncomeRepo implementa repository.IncomeRepository; o balance usa apenas
// SumByUserAndCompetence — os demais métodos entram em pânico se chamados.
type stubIncomeRepo struct {
	SumFn func(ctx context.Context, userID uuid.UUID, competence string) (decimal.Decimal, error)
}

var _ repository.IncomeRepository = (*stubIncomeRepo)(nil)

func (s *stubIncomeRepo) SumByUserAndCompetence(ctx context.Context, userID uuid.UUID, competence string) (decimal.Decimal, error) {
	return s.SumFn(ctx, userID, competence)
}
func (s *stubIncomeRepo) Create(context.Context, *entity.Income) error { panic("unexpected Create") }
func (s *stubIncomeRepo) FindByID(context.Context, uuid.UUID) (*entity.Income, error) {
	panic("unexpected FindByID")
}
func (s *stubIncomeRepo) ListByUserAndCompetence(context.Context, uuid.UUID, string) ([]entity.Income, error) {
	panic("unexpected List")
}
func (s *stubIncomeRepo) Update(context.Context, *entity.Income) error { panic("unexpected Update") }
func (s *stubIncomeRepo) Delete(context.Context, uuid.UUID) error      { panic("unexpected Delete") }
func (s *stubIncomeRepo) ListRecurrentByCompetence(context.Context, string) ([]entity.Income, error) {
	panic("unexpected ListRecurrent")
}
func (s *stubIncomeRepo) ExistsByOriginAndCompetence(context.Context, uuid.UUID, string) (bool, error) {
	panic("unexpected Exists")
}

// stubBudgetRepo implementa repository.GlobalBudgetRepository; o balance usa
// apenas GetByUserAndCompetence.
type stubBudgetRepo struct {
	GetByUserAndCompetenceFn func(ctx context.Context, userID uuid.UUID, competence string) (*entity.GlobalBudget, error)
}

var _ repository.GlobalBudgetRepository = (*stubBudgetRepo)(nil)

func (s *stubBudgetRepo) GetByUserAndCompetence(ctx context.Context, userID uuid.UUID, competence string) (*entity.GlobalBudget, error) {
	return s.GetByUserAndCompetenceFn(ctx, userID, competence)
}
func (s *stubBudgetRepo) Create(context.Context, *entity.GlobalBudget) error {
	panic("unexpected Create")
}
func (s *stubBudgetRepo) GetByID(context.Context, uuid.UUID) (*entity.GlobalBudget, error) {
	panic("unexpected GetByID")
}
func (s *stubBudgetRepo) Update(context.Context, *entity.GlobalBudget) error {
	panic("unexpected Update")
}
func (s *stubBudgetRepo) ExistsByUserAndCompetence(context.Context, uuid.UUID, string) (bool, error) {
	panic("unexpected Exists")
}
func (s *stubBudgetRepo) Upsert(context.Context, *entity.GlobalBudget) error {
	panic("unexpected Upsert")
}
func (s *stubBudgetRepo) Delete(context.Context, uuid.UUID) error { panic("unexpected Delete") }

// stubTxClient implementa httpclient.TransactionClient; o balance usa
// GetExpenseTotals e GetAccountsPayable.
type stubTxClient struct {
	GetExpenseTotalsFn   func(ctx context.Context, userID, competence string) (*httpclient.ExpenseTotalsOutput, error)
	GetAccountsPayableFn func(ctx context.Context, userID, dueDateUntil string) ([]httpclient.PendingBillOutput, error)
}

var _ httpclient.TransactionClient = (*stubTxClient)(nil)

func (s *stubTxClient) GetExpenseTotals(ctx context.Context, userID, competence string) (*httpclient.ExpenseTotalsOutput, error) {
	return s.GetExpenseTotalsFn(ctx, userID, competence)
}
func (s *stubTxClient) GetAccountsPayable(ctx context.Context, userID, dueDateUntil string) ([]httpclient.PendingBillOutput, error) {
	return s.GetAccountsPayableFn(ctx, userID, dueDateUntil)
}
func (s *stubTxClient) GetExpensesByCategory(context.Context, string, string, string) (*httpclient.ExpensesByCategoryOutput, error) {
	panic("unexpected GetExpensesByCategory")
}

// happyFixture monta o cenário base do FDD-004 §5: receita 7500, gasto 3200
// (1800 pessoal + 1400 compartilhado), conta pendente 850, teto 5000.
func happyFixture() (*stubIncomeRepo, *stubBudgetRepo, *stubTxClient) {
	incomeRepo := &stubIncomeRepo{
		SumFn: func(context.Context, uuid.UUID, string) (decimal.Decimal, error) {
			return decimal.RequireFromString("7500.00"), nil
		},
	}
	budgetRepo := &stubBudgetRepo{
		GetByUserAndCompetenceFn: func(_ context.Context, userID uuid.UUID, competence string) (*entity.GlobalBudget, error) {
			return &entity.GlobalBudget{
				ID:         uuid.New(),
				UserID:     userID,
				Competence: competence,
				Ceiling:    decimal.RequireFromString("5000.00"),
			}, nil
		},
	}
	client := &stubTxClient{
		GetExpenseTotalsFn: func(_ context.Context, _, competence string) (*httpclient.ExpenseTotalsOutput, error) {
			return &httpclient.ExpenseTotalsOutput{
				Competence:    competence,
				TotalPersonal: decimal.RequireFromString("1800.00"),
				TotalShared:   decimal.RequireFromString("1400.00"),
				TotalGeneral:  decimal.RequireFromString("3200.00"),
			}, nil
		},
		GetAccountsPayableFn: func(context.Context, string, string) ([]httpclient.PendingBillOutput, error) {
			return []httpclient.PendingBillOutput{
				{ID: uuid.NewString(), Description: "Escola", Amount: decimal.RequireFromString("850.00"), Status: "PENDING"},
			}, nil
		},
	}
	return incomeRepo, budgetRepo, client
}
