package income

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
	repo   repository.IncomeRepository
	logger *slog.Logger
}

func NewDeleteUseCase(repo repository.IncomeRepository, logger *slog.Logger) DeleteUseCase {
	return &deleteUseCase{repo: repo, logger: logger}
}

func (uc *deleteUseCase) Execute(ctx context.Context, id uuid.UUID) error {
	start := time.Now()
	uc.logger.InfoContext(ctx, "income.delete started",
		slog.String("operation", "income.delete"),
		slog.String("income_id", id.String()),
	)

	if err := uc.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrIncomeNotFound
		}
		return err
	}

	uc.logger.InfoContext(ctx, "income.delete completed",
		slog.String("operation", "income.delete"),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.String("income_id", id.String()),
	)
	return nil
}
