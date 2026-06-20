package income

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
)

type UpdateUseCase interface {
	Execute(ctx context.Context, input UpdateInput) (*IncomeOutput, error)
}

type updateUseCase struct {
	repo   repository.IncomeRepository
	logger *slog.Logger
}

func NewUpdateUseCase(repo repository.IncomeRepository, logger *slog.Logger) UpdateUseCase {
	return &updateUseCase{repo: repo, logger: logger}
}

func (uc *updateUseCase) Execute(ctx context.Context, input UpdateInput) (*IncomeOutput, error) {
	start := time.Now()
	uc.logger.InfoContext(ctx, "income.update started",
		slog.String("operation", "income.update"),
		slog.String("income_id", input.ID.String()),
	)

	income, err := uc.repo.FindByID(ctx, input.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrIncomeNotFound
		}
		return nil, err
	}

	// REC-03: registro propagado (origin_id != nil) é imutável.
	if income.OriginID != nil {
		logRuleViolation(ctx, uc.logger, "REC-03", "cannot edit propagated income")
		return nil, ErrCannotEditPropagatedIncome
	}

	// Aplica apenas os campos editáveis fornecidos.
	if input.Description != nil {
		income.Description = *input.Description
	}
	if input.Amount != nil {
		if !input.Amount.IsPositive() {
			logRuleViolation(ctx, uc.logger, "REC-02", "amount must be greater than zero")
			return nil, ErrInvalidAmount
		}
		income.Amount = *input.Amount
	}
	if input.Type != nil {
		if !isValidIncomeType(*input.Type) {
			logRuleViolation(ctx, uc.logger, "REC-01", "invalid income type")
			return nil, ErrInvalidIncomeType
		}
		income.Type = *input.Type
	}
	if input.Date != nil {
		date, err := parseDate(*input.Date)
		if err != nil {
			logRuleViolation(ctx, uc.logger, "REC-01", "invalid date format")
			return nil, ErrInvalidDate
		}
		income.Date = date
	}
	if input.Recurrent != nil {
		income.Recurrent = *input.Recurrent
	}

	if err := uc.repo.Update(ctx, income); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "income.update completed",
		slog.String("operation", "income.update"),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.String("income_id", income.ID.String()),
	)
	out := toOutput(income)
	return &out, nil
}
