package goal

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
)

type ListUseCase interface {
	Execute(ctx context.Context, input ListInput) ([]GoalOutput, error)
}

type listUseCase struct {
	repo   repository.ReductionGoalRepository
	logger *slog.Logger
}

func NewListUseCase(repo repository.ReductionGoalRepository, logger *slog.Logger) ListUseCase {
	return &listUseCase{repo: repo, logger: logger}
}

func (uc *listUseCase) Execute(ctx context.Context, input ListInput) ([]GoalOutput, error) {
	start := time.Now()
	uc.logger.InfoContext(ctx, "goal.list started",
		slog.String("operation", "goal.list"),
		slog.String("user_id", input.UserID.String()),
		slog.String("competence", input.Competence),
	)

	if input.UserID == uuid.Nil {
		logRuleViolation(ctx, uc.logger, "MET-04", "missing user_id")
		return nil, ErrMissingRequiredField
	}
	if !isValidCompetence(input.Competence) {
		logRuleViolation(ctx, uc.logger, "MET-04", "invalid competence format")
		return nil, ErrInvalidCompetence
	}

	goals, err := uc.repo.ListByUserAndCompetence(ctx, input.UserID, input.Competence)
	if err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "goal.list completed",
		slog.String("operation", "goal.list"),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.Int("count", len(goals)),
	)
	return toOutputs(goals), nil
}
