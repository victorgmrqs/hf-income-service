package global_budget

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
	"github.com/victorgmrqs/hf-income-service/src/pkg/httpclient"
)

type PreviewNextUseCase interface {
	Execute(ctx context.Context, input PreviewInput) (*PreviewOutput, error)
}

type previewNextUseCase struct {
	repo   repository.GlobalBudgetRepository
	client httpclient.TransactionClient
	logger *slog.Logger
	nowFn  func() time.Time // injetável nos testes; a competência corrente vem do relógio
}

func NewPreviewNextUseCase(
	repo repository.GlobalBudgetRepository,
	client httpclient.TransactionClient,
	logger *slog.Logger,
) PreviewNextUseCase {
	return &previewNextUseCase{repo: repo, client: client, logger: logger, nowFn: time.Now}
}

// Execute calcula o teto sugerido para a próxima competência sem persistir
// (ORC-03/04, FDD-002 §4) — mesma lógica do auto-ajuste, base = competência corrente.
func (uc *previewNextUseCase) Execute(ctx context.Context, input PreviewInput) (*PreviewOutput, error) {
	start := time.Now()
	current := currentCompetence(uc.nowFn())
	uc.logger.InfoContext(ctx, "global_budget.preview_next started",
		slog.String("operation", "global_budget.preview_next"),
		slog.String("user_id", input.UserID.String()),
		slog.String("competence", current),
	)

	// ORC-01: campos obrigatórios.
	if input.UserID == uuid.Nil {
		logRuleViolation(ctx, uc.logger, "ORC-01", "missing required field")
		return nil, ErrMissingRequiredField
	}

	// ORC-03: sem teto na competência corrente não há base para o preview.
	currentBudget, err := uc.repo.GetByUserAndCompetence(ctx, input.UserID, current)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logRuleViolation(ctx, uc.logger, "ORC-03", "no current budget for preview base")
			return nil, ErrNoPreviousBudget
		}
		return nil, err
	}

	totals, err := uc.client.GetExpenseTotals(ctx, input.UserID.String(), current)
	if err != nil {
		uc.logger.ErrorContext(ctx, "global_budget.preview_next upstream failure",
			slog.String("trace_id", traceIDFromContext(ctx)),
			slog.String("upstream", "hf-transaction-service"),
			slog.String("error", err.Error()),
			slog.String("user_id", input.UserID.String()),
			slog.String("competence", current),
		)
		return nil, ErrUpstreamUnavailable
	}

	spending := totals.TotalGeneral
	suggested := currentBudget.Ceiling
	reason := AdjustmentReasonSpendingEqualsCeiling
	if spending.LessThan(currentBudget.Ceiling) {
		// ORC-03: gasto < teto → sugestão de aperto.
		suggested = spending
		reason = AdjustmentReasonSpendingBelowCeiling
	}
	// ORC-04: gasto >= teto → sugestão mantém o teto corrente.

	next, err := nextCompetence(current)
	if err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "global_budget.preview_next completed",
		slog.String("operation", "global_budget.preview_next"),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.String("adjustment_reason", reason),
	)
	return &PreviewOutput{
		CurrentCompetence: current,
		CurrentCeiling:    currentBudget.Ceiling,
		CurrentSpending:   spending,
		NextCompetence:    next,
		SuggestedCeiling:  suggested,
		AdjustmentReason:  reason,
	}, nil
}
