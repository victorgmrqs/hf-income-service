package global_budget

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
)

type GetUseCase interface {
	Execute(ctx context.Context, input GetInput) (*GlobalBudgetOutput, error)
}

type getUseCase struct {
	repo   repository.GlobalBudgetRepository
	logger *slog.Logger
}

func NewGetUseCase(repo repository.GlobalBudgetRepository, logger *slog.Logger) GetUseCase {
	return &getUseCase{repo: repo, logger: logger}
}

func (uc *getUseCase) Execute(ctx context.Context, input GetInput) (*GlobalBudgetOutput, error) {
	start := time.Now()
	uc.logger.InfoContext(ctx, "global_budget.get started",
		slog.String("operation", "global_budget.get"),
		slog.String("user_id", input.UserID.String()),
		slog.String("competence", input.Competence),
	)

	// ORC-01: consulta por (user_id, competence).
	if input.UserID == uuid.Nil || input.Competence == "" {
		logRuleViolation(ctx, uc.logger, "ORC-01", "missing required field")
		return nil, ErrMissingRequiredField
	}

	budget, err := uc.repo.GetByUserAndCompetence(ctx, input.UserID, input.Competence)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBudgetNotFound
		}
		return nil, err
	}

	uc.logger.InfoContext(ctx, "global_budget.get completed",
		slog.String("operation", "global_budget.get"),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.String("budget_id", budget.ID.String()),
	)
	out := toOutput(budget)
	return &out, nil
}
