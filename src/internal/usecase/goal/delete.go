package goal

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
)

type DeleteUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) error
}

type deleteUseCase struct {
	repo   repository.ReductionGoalRepository
	logger *slog.Logger
}

func NewDeleteUseCase(repo repository.ReductionGoalRepository, logger *slog.Logger) DeleteUseCase {
	return &deleteUseCase{repo: repo, logger: logger}
}

func (uc *deleteUseCase) Execute(ctx context.Context, id uuid.UUID) error {
	start := time.Now()
	uc.logger.InfoContext(ctx, "goal.delete started",
		slog.String("operation", "goal.delete"),
		slog.String("goal_id", id.String()),
	)

	if err := uc.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGoalNotFound
		}
		return err
	}

	uc.logger.InfoContext(ctx, "goal.delete completed",
		slog.String("operation", "goal.delete"),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.String("goal_id", id.String()),
	)
	return nil
}
