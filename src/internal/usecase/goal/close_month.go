package goal

import (
	"context"
	"log/slog"
	"time"

	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
	"github.com/victorgmrqs/hf-income-service/src/pkg/httpclient"
)

type CloseMonthUseCase interface {
	Execute(ctx context.Context, input CloseMonthInput) (*CloseMonthOutput, error)
}

type closeMonthUseCase struct {
	repo   repository.ReductionGoalRepository
	client httpclient.TransactionClient
	logger *slog.Logger
}

func NewCloseMonthUseCase(
	repo repository.ReductionGoalRepository,
	client httpclient.TransactionClient,
	logger *slog.Logger,
) CloseMonthUseCase {
	return &closeMonthUseCase{repo: repo, client: client, logger: logger}
}

// Execute fecha a competência (MET-06, FDD-003 §4): para cada meta ainda aberta
// (achieved null) busca o gasto final da categoria e grava
// achieved = gasto ≤ meta. Falha de upstream ou de persistência pula a meta com
// log ERROR e segue para as demais; retorna a contagem de metas fechadas.
func (uc *closeMonthUseCase) Execute(ctx context.Context, input CloseMonthInput) (*CloseMonthOutput, error) {
	start := time.Now()
	uc.logger.InfoContext(ctx, "goal.close_month started",
		slog.String("operation", "goal.close_month"),
		slog.String("competence", input.Competence),
	)

	// MET-06: competência obrigatória.
	if input.Competence == "" {
		logRuleViolation(ctx, uc.logger, "MET-06", "missing required field")
		return nil, ErrMissingRequiredField
	}
	// MET-06: competence YYYY-MM.
	if !isValidCompetence(input.Competence) {
		logRuleViolation(ctx, uc.logger, "MET-06", "invalid competence format")
		return nil, ErrInvalidCompetence
	}

	goals, err := uc.repo.ListWithNullAchievedByCompetence(ctx, input.Competence)
	if err != nil {
		return nil, err
	}

	closed := 0
	for i := range goals {
		goal := &goals[i]
		result, err := uc.client.GetExpensesByCategory(ctx, goal.UserID.String(), goal.CategoryID.String(), input.Competence)
		if err != nil || result == nil {
			reason := "upstream returned no data"
			if err != nil {
				reason = err.Error()
			}
			uc.logger.ErrorContext(ctx, "goal.close_month skipped goal",
				slog.String("trace_id", traceIDFromContext(ctx)),
				slog.String("goal_id", goal.ID.String()),
				slog.String("category_id", goal.CategoryID.String()),
				slog.String("competence", input.Competence),
				slog.String("reason", reason),
			)
			continue
		}

		// MET-06: achieved = gasto final ≤ meta.
		achieved := result.Total.LessThanOrEqual(goal.TargetAmount)
		if err := uc.repo.SetAchieved(ctx, goal.ID, achieved); err != nil {
			uc.logger.ErrorContext(ctx, "goal.close_month skipped goal",
				slog.String("trace_id", traceIDFromContext(ctx)),
				slog.String("goal_id", goal.ID.String()),
				slog.String("category_id", goal.CategoryID.String()),
				slog.String("competence", input.Competence),
				slog.String("reason", err.Error()),
			)
			continue
		}
		closed++
	}

	uc.logger.InfoContext(ctx, "goal.close_month completed",
		slog.String("operation", "goal.close_month"),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.Int("closed", closed),
	)
	return &CloseMonthOutput{Closed: closed}, nil
}
