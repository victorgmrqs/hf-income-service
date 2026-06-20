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

type GetUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) (*IncomeOutput, error)
}

type getUseCase struct {
	repo   repository.IncomeRepository
	logger *slog.Logger
}

func NewGetUseCase(repo repository.IncomeRepository, logger *slog.Logger) GetUseCase {
	return &getUseCase{repo: repo, logger: logger}
}

func (uc *getUseCase) Execute(ctx context.Context, id uuid.UUID) (*IncomeOutput, error) {
	start := time.Now()
	uc.logger.InfoContext(ctx, "income.get started",
		slog.String("operation", "income.get"),
		slog.String("income_id", id.String()),
	)

	income, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrIncomeNotFound
		}
		return nil, err
	}

	uc.logger.InfoContext(ctx, "income.get completed",
		slog.String("operation", "income.get"),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.String("income_id", id.String()),
	)
	out := toOutput(income)
	return &out, nil
}
