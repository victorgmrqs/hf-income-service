package goal

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
)

type UpdateUseCase interface {
	Execute(ctx context.Context, input UpdateInput) (*GoalOutput, error)
}

type updateUseCase struct {
	repo   repository.ReductionGoalRepository
	logger *slog.Logger
}

func NewUpdateUseCase(repo repository.ReductionGoalRepository, logger *slog.Logger) UpdateUseCase {
	return &updateUseCase{repo: repo, logger: logger}
}

// Execute atualiza uma meta de redução. Apenas target_amount é editável — os
// demais campos (user_id, category_id, competence, previous_amount, achieved)
// permanecem imutáveis (MET-04, FDD-003 §5).
func (uc *updateUseCase) Execute(ctx context.Context, input UpdateInput) (*GoalOutput, error) {
	start := time.Now()
	uc.logger.InfoContext(ctx, "goal.update started",
		slog.String("operation", "goal.update"),
		slog.String("goal_id", input.ID.String()),
	)

	goal, err := uc.repo.GetByID(ctx, input.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGoalNotFound
		}
		return nil, err
	}

	// MET-04: target_amount > 0.
	if !input.TargetAmount.IsPositive() {
		logRuleViolation(ctx, uc.logger, "MET-04", "target amount must be greater than zero")
		return nil, ErrInvalidTargetAmount
	}

	goal.TargetAmount = input.TargetAmount

	if err := uc.repo.Update(ctx, goal); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "goal.update completed",
		slog.String("operation", "goal.update"),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.String("goal_id", goal.ID.String()),
	)
	out := toOutput(goal)
	return &out, nil
}
