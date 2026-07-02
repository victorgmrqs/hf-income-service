package global_budget

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
)

type CreateUseCase interface {
	Execute(ctx context.Context, input CreateInput) (*GlobalBudgetOutput, error)
}

type createUseCase struct {
	repo   repository.GlobalBudgetRepository
	logger *slog.Logger
}

func NewCreateUseCase(repo repository.GlobalBudgetRepository, logger *slog.Logger) CreateUseCase {
	return &createUseCase{repo: repo, logger: logger}
}

func (uc *createUseCase) Execute(ctx context.Context, input CreateInput) (*GlobalBudgetOutput, error) {
	start := time.Now()
	uc.logger.InfoContext(ctx, "global_budget.create started",
		slog.String("operation", "global_budget.create"),
		slog.String("user_id", input.UserID.String()),
		slog.String("competence", input.Competence),
	)

	// ORC-01: campos obrigatórios.
	if input.UserID == uuid.Nil || input.Competence == "" {
		logRuleViolation(ctx, uc.logger, "ORC-01", "missing required field")
		return nil, ErrMissingRequiredField
	}
	// ORC-01: ceiling > 0.
	if !input.Ceiling.IsPositive() {
		logRuleViolation(ctx, uc.logger, "ORC-01", "ceiling must be greater than zero")
		return nil, ErrInvalidCeiling
	}
	// ORC-01: competence YYYY-MM.
	if !isValidCompetence(input.Competence) {
		logRuleViolation(ctx, uc.logger, "ORC-01", "invalid competence format")
		return nil, ErrInvalidCompetence
	}

	// ORC-01: no máximo um teto por (user_id, competence).
	existing, err := uc.repo.GetByUserAndCompetence(ctx, input.UserID, input.Competence)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		logRuleViolation(ctx, uc.logger, "ORC-01", "budget already exists for competence")
		return nil, ErrBudgetAlreadyExists
	}

	budget := &entity.GlobalBudget{
		UserID:       input.UserID,
		Competence:   input.Competence,
		Ceiling:      input.Ceiling,
		AutoAdjusted: false,
	}
	if err := uc.repo.Create(ctx, budget); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "global_budget.create completed",
		slog.String("operation", "global_budget.create"),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.String("budget_id", budget.ID.String()),
		slog.Bool("auto_adjusted", budget.AutoAdjusted),
	)
	out := toOutput(budget)
	return &out, nil
}
