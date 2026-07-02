package global_budget

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
)

type UpdateUseCase interface {
	Execute(ctx context.Context, input UpdateInput) (*GlobalBudgetOutput, error)
}

type updateUseCase struct {
	repo   repository.GlobalBudgetRepository
	logger *slog.Logger
}

func NewUpdateUseCase(repo repository.GlobalBudgetRepository, logger *slog.Logger) UpdateUseCase {
	return &updateUseCase{repo: repo, logger: logger}
}

func (uc *updateUseCase) Execute(ctx context.Context, input UpdateInput) (*GlobalBudgetOutput, error) {
	start := time.Now()
	uc.logger.InfoContext(ctx, "global_budget.update started",
		slog.String("operation", "global_budget.update"),
		slog.String("budget_id", input.ID.String()),
	)

	budget, err := uc.repo.GetByID(ctx, input.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBudgetNotFound
		}
		return nil, err
	}

	// ORC-01: ceiling > 0.
	if !input.Ceiling.IsPositive() {
		logRuleViolation(ctx, uc.logger, "ORC-01", "ceiling must be greater than zero")
		return nil, ErrInvalidCeiling
	}

	// ORC-05: edição manual sempre força auto_adjusted = false.
	budget.Ceiling = input.Ceiling
	budget.AutoAdjusted = false

	if err := uc.repo.Update(ctx, budget); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "global_budget.update completed",
		slog.String("operation", "global_budget.update"),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.String("budget_id", budget.ID.String()),
		slog.Bool("auto_adjusted", budget.AutoAdjusted),
	)
	out := toOutput(budget)
	return &out, nil
}
