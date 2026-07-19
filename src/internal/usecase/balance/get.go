package balance

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
	"github.com/victorgmrqs/hf-income-service/src/pkg/httpclient"
)

type GetUseCase interface {
	Execute(ctx context.Context, input GetInput) (*BalanceOutput, error)
}

type getUseCase struct {
	incomeRepo repository.IncomeRepository
	budgetRepo repository.GlobalBudgetRepository
	client     httpclient.TransactionClient
	logger     *slog.Logger
	tracer     trace.Tracer
}

func NewGetUseCase(
	incomeRepo repository.IncomeRepository,
	budgetRepo repository.GlobalBudgetRepository,
	client httpclient.TransactionClient,
	logger *slog.Logger,
) GetUseCase {
	return &getUseCase{
		incomeRepo: incomeRepo,
		budgetRepo: budgetRepo,
		client:     client,
		logger:     logger,
		tracer:     otel.Tracer("hf-income-service/balance"),
	}
}

// Execute calcula o saldo mensal sob demanda (FDD-004 §4): agrega 4 fontes em
// paralelo via errgroup — receitas e teto (locais), despesas realizadas e contas
// a pagar pendentes (hf-transaction-service). Qualquer falha cancela o contexto
// e retorna erro imediato — SAL nunca retorna dados parciais.
func (uc *getUseCase) Execute(ctx context.Context, input GetInput) (*BalanceOutput, error) {
	start := time.Now()
	ctx, span := uc.tracer.Start(ctx, "balance.get")
	defer span.End()

	uc.logger.InfoContext(ctx, "balance.get started",
		slog.String("operation", "balance.get"),
		slog.String("user_id", input.UserID.String()),
		slog.String("competence", input.Competence),
	)

	// SAL-01: campos obrigatórios.
	if input.UserID == uuid.Nil || input.Competence == "" {
		logRuleViolation(ctx, uc.logger, "SAL-01", "missing required field")
		return nil, ErrMissingRequiredField
	}
	// SAL-01: competence YYYY-MM.
	if !isValidCompetence(input.Competence) {
		logRuleViolation(ctx, uc.logger, "SAL-01", "invalid competence format")
		return nil, ErrInvalidCompetence
	}
	lastDay, err := httpclient.LastDayOfMonth(input.Competence)
	if err != nil {
		logRuleViolation(ctx, uc.logger, "SAL-01", "invalid competence format")
		return nil, ErrInvalidCompetence
	}

	var (
		totalIncome decimal.Decimal
		budget      *entity.GlobalBudget
		totals      *httpclient.ExpenseTotalsOutput
		bills       []httpclient.PendingBillOutput
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		sum, err := uc.incomeRepo.SumByUserAndCompetence(gctx, input.UserID, input.Competence)
		if err != nil {
			return fmt.Errorf("sum incomes: %w", err)
		}
		totalIncome = sum
		return nil
	})
	g.Go(func() error {
		b, err := uc.budgetRepo.GetByUserAndCompetence(gctx, input.UserID, input.Competence)
		if err != nil {
			// Sem teto cadastrado não é erro: campos ORC saem null/false (ORC-06/07).
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return fmt.Errorf("get global budget: %w", err)
		}
		budget = b
		return nil
	})
	g.Go(func() error {
		sctx, sspan := uc.tracer.Start(gctx, "balance.get.expense_totals")
		defer sspan.End()
		t, err := uc.client.GetExpenseTotals(sctx, input.UserID.String(), input.Competence)
		if err != nil {
			return uc.mapUpstreamErr(ctx, "/expenses/user/{id}/totals", err, input)
		}
		totals = t
		return nil
	})
	g.Go(func() error {
		sctx, sspan := uc.tracer.Start(gctx, "balance.get.accounts_payable")
		defer sspan.End()
		b, err := uc.client.GetAccountsPayable(sctx, input.UserID.String(), lastDay)
		if err != nil {
			return uc.mapUpstreamErr(ctx, "/accounts-payable", err, input)
		}
		bills = b
		return nil
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// SAL-02: despesas comprometidas = soma das contas PENDING da competência.
	committed := decimal.Zero
	for _, bill := range bills {
		committed = committed.Add(bill.Amount)
	}

	// SAL-01/03/05.
	balanceToday := totalIncome.Sub(totals.TotalGeneral)
	projected := balanceToday.Sub(committed)

	out := &BalanceOutput{
		UserID:              input.UserID,
		Competence:          input.Competence,
		TotalIncome:         totalIncome,
		TotalPersonal:       totals.TotalPersonal,
		TotalShared:         totals.TotalShared,
		TotalExpenses:       totals.TotalGeneral,
		BalanceToday:        balanceToday,
		CommittedBills:      committed,
		ProjectedBalance:    projected,
		IsProjectedNegative: projected.IsNegative(),
	}
	// ORC-06/07: campos de teto apenas quando cadastrado.
	if budget != nil {
		ceiling := budget.Ceiling
		out.Ceiling = &ceiling
		out.CeilingExceeded = totals.TotalGeneral.GreaterThan(ceiling)
		// Teto 0 é possível via auto-ajuste com gasto zero (FDD-002 §4):
		// percentual indefinido — permanece null; exceeded cobre o estouro.
		if !ceiling.IsZero() {
			pct := int(totals.TotalGeneral.Div(ceiling).Mul(decimal.NewFromInt(100)).IntPart())
			out.CeilingUsagePct = &pct
		}
	}

	uc.logger.InfoContext(ctx, "balance.get completed",
		slog.String("operation", "balance.get"),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.String("balance_today", out.BalanceToday.String()),
		slog.String("projected_balance", out.ProjectedBalance.String()),
		slog.Bool("is_projected_negative", out.IsProjectedNegative),
	)
	return out, nil
}

// mapUpstreamErr loga a falha upstream com contexto completo (FDD-004 §7) e
// converte os erros sentinela do httpclient nos erros de domínio do SAL
// (timeout → 503 UPSTREAM_TIMEOUT; status inesperado → 502 UPSTREAM_ERROR).
func (uc *getUseCase) mapUpstreamErr(ctx context.Context, endpoint string, err error, input GetInput) error {
	uc.logger.ErrorContext(ctx, "balance.get upstream failure",
		slog.String("trace_id", traceIDFromContext(ctx)),
		slog.String("upstream", "hf-transaction-service"),
		slog.String("upstream_endpoint", endpoint),
		slog.String("error", err.Error()),
		slog.String("user_id", input.UserID.String()),
		slog.String("competence", input.Competence),
	)
	switch {
	case errors.Is(err, httpclient.ErrUpstreamTimeout):
		return ErrUpstreamTimeout
	case errors.Is(err, httpclient.ErrUpstreamError):
		return ErrUpstreamError
	default:
		return err
	}
}
