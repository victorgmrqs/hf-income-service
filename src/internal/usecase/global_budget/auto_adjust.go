package global_budget

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
	"github.com/victorgmrqs/hf-income-service/src/pkg/httpclient"
)

type AutoAdjustUseCase interface {
	Execute(ctx context.Context, input AutoAdjustInput) (*GlobalBudgetOutput, error)
}

type autoAdjustUseCase struct {
	repo   repository.GlobalBudgetRepository
	client httpclient.TransactionClient
	logger *slog.Logger
}

func NewAutoAdjustUseCase(
	repo repository.GlobalBudgetRepository,
	client httpclient.TransactionClient,
	logger *slog.Logger,
) AutoAdjustUseCase {
	return &autoAdjustUseCase{repo: repo, client: client, logger: logger}
}

// Execute aplica o auto-ajuste progressivo do teto (ORC-03/04/05, FDD-002 §4):
// busca o teto da competência anterior, obtém o gasto real no hf-transaction-service
// e faz upsert do teto da competência de destino — idempotente para re-execuções.
func (uc *autoAdjustUseCase) Execute(ctx context.Context, input AutoAdjustInput) (*GlobalBudgetOutput, error) {
	start := time.Now()
	uc.logger.InfoContext(ctx, "global_budget.auto_adjust started",
		slog.String("operation", "global_budget.auto_adjust"),
		slog.String("user_id", input.UserID.String()),
		slog.String("competence", input.Competence),
	)

	// ORC-01: campos obrigatórios.
	if input.UserID == uuid.Nil || input.Competence == "" {
		logRuleViolation(ctx, uc.logger, "ORC-01", "missing required field")
		return nil, ErrMissingRequiredField
	}
	// ORC-01: competence YYYY-MM (destino do ajuste).
	if !isValidCompetence(input.Competence) {
		logRuleViolation(ctx, uc.logger, "ORC-01", "invalid competence format")
		return nil, ErrInvalidCompetence
	}

	prevCompetence, err := previousCompetence(input.Competence)
	if err != nil {
		logRuleViolation(ctx, uc.logger, "ORC-01", "invalid competence format")
		return nil, ErrInvalidCompetence
	}

	// ORC-03: sem teto anterior não há base para o ajuste.
	prevBudget, err := uc.repo.GetByUserAndCompetence(ctx, input.UserID, prevCompetence)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logRuleViolation(ctx, uc.logger, "ORC-03", "no previous budget for auto-adjust base")
			return nil, ErrNoPreviousBudget
		}
		return nil, err
	}

	totals, err := uc.client.GetExpenseTotals(ctx, input.UserID.String(), prevCompetence)
	if err != nil {
		// FDD-002 §6: falha upstream → 503, log ERROR com contexto completo
		// (status code vem embutido na mensagem de erro do httpclient).
		uc.logger.ErrorContext(ctx, "global_budget.auto_adjust upstream failure",
			slog.String("trace_id", traceIDFromContext(ctx)),
			slog.String("upstream", "hf-transaction-service"),
			slog.String("error", err.Error()),
			slog.String("user_id", input.UserID.String()),
			slog.String("competence", prevCompetence),
		)
		return nil, ErrUpstreamUnavailable
	}

	spending := totals.TotalGeneral
	// FDD-002 §4 (fluxo alternativo): sem gastos no mês anterior → teto 0, caso de atenção.
	if spending.IsZero() {
		uc.logger.WarnContext(ctx, "auto-adjust with zero previous spending",
			slog.String("trace_id", traceIDFromContext(ctx)),
			slog.String("rule_id", "ORC-03"),
			slog.String("reason", "previous spending is zero; new ceiling will be zero"),
			slog.String("user_id", input.UserID.String()),
			slog.String("competence", prevCompetence),
		)
	}

	newCeiling := prevBudget.Ceiling
	autoAdjusted := false
	if spending.LessThan(prevBudget.Ceiling) {
		// ORC-03: gasto < teto anterior → aperto progressivo, auditável.
		newCeiling = spending
		autoAdjusted = true
	}
	// ORC-04: gasto >= teto anterior → teto mantido, auto_adjusted permanece false.

	budget := &entity.GlobalBudget{
		UserID:       input.UserID,
		Competence:   input.Competence,
		Ceiling:      newCeiling,
		AutoAdjusted: autoAdjusted,
	}
	// ORC-05: upsert — re-execução do job na mesma competência é idempotente.
	if err := uc.repo.Upsert(ctx, budget); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "global_budget.auto_adjust completed",
		slog.String("operation", "global_budget.auto_adjust"),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.String("budget_id", budget.ID.String()),
		slog.Bool("auto_adjusted", budget.AutoAdjusted),
	)
	out := toOutput(budget)
	return &out, nil
}
