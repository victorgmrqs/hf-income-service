package income

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
)

type ListInput struct {
	UserID     uuid.UUID
	Competence string
}

type ListUseCase interface {
	Execute(ctx context.Context, input ListInput) (*ListOutput, error)
}

type listUseCase struct {
	repo   repository.IncomeRepository
	logger *slog.Logger
}

func NewListUseCase(repo repository.IncomeRepository, logger *slog.Logger) ListUseCase {
	return &listUseCase{repo: repo, logger: logger}
}

func (uc *listUseCase) Execute(ctx context.Context, input ListInput) (*ListOutput, error) {
	start := time.Now()
	uc.logger.InfoContext(ctx, "income.list started",
		slog.String("operation", "income.list"),
		slog.String("user_id", input.UserID.String()),
		slog.String("competence", input.Competence),
	)

	if input.UserID == uuid.Nil {
		logRuleViolation(ctx, uc.logger, "REC-01", "missing user_id")
		return nil, ErrMissingRequiredField
	}
	if !isValidCompetence(input.Competence) {
		logRuleViolation(ctx, uc.logger, "REC-01", "invalid competence format")
		return nil, ErrInvalidCompetence
	}

	items, err := uc.repo.ListByUserAndCompetence(ctx, input.UserID, input.Competence)
	if err != nil {
		return nil, err
	}

	// REC-06: total_income = soma das receitas do usuário na competência.
	total := decimal.Zero
	for i := range items {
		total = total.Add(items[i].Amount)
	}

	uc.logger.InfoContext(ctx, "income.list completed",
		slog.String("operation", "income.list"),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.Int("count", len(items)),
	)
	return &ListOutput{
		Competence:  input.Competence,
		TotalIncome: total,
		Items:       toOutputs(items),
	}, nil
}
