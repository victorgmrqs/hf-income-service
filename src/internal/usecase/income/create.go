package income

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
)

type CreateUseCase interface {
	Execute(ctx context.Context, input CreateInput) (*IncomeOutput, error)
}

type createUseCase struct {
	repo   repository.IncomeRepository
	logger *slog.Logger
}

func NewCreateUseCase(repo repository.IncomeRepository, logger *slog.Logger) CreateUseCase {
	return &createUseCase{repo: repo, logger: logger}
}

func (uc *createUseCase) Execute(ctx context.Context, input CreateInput) (*IncomeOutput, error) {
	start := time.Now()
	uc.logger.InfoContext(ctx, "income.create started",
		slog.String("operation", "income.create"),
		slog.String("user_id", input.UserID.String()),
		slog.String("competence", input.Competence),
	)

	// REC-01: campos obrigatórios.
	if input.UserID == uuid.Nil || input.Description == "" || input.Date == "" ||
		input.Competence == "" || input.Type == "" {
		logRuleViolation(ctx, uc.logger, "REC-01", "missing required field")
		return nil, ErrMissingRequiredField
	}
	// REC-02: amount > 0.
	if !input.Amount.IsPositive() {
		logRuleViolation(ctx, uc.logger, "REC-02", "amount must be greater than zero")
		return nil, ErrInvalidAmount
	}
	// REC-01: type válido.
	if !isValidIncomeType(input.Type) {
		logRuleViolation(ctx, uc.logger, "REC-01", "invalid income type")
		return nil, ErrInvalidIncomeType
	}
	// REC-01: competence YYYY-MM.
	if !isValidCompetence(input.Competence) {
		logRuleViolation(ctx, uc.logger, "REC-01", "invalid competence format")
		return nil, ErrInvalidCompetence
	}
	date, err := parseDate(input.Date)
	if err != nil {
		logRuleViolation(ctx, uc.logger, "REC-01", "invalid date format")
		return nil, ErrInvalidDate
	}

	income := &entity.Income{
		UserID:      input.UserID,
		Description: input.Description,
		Amount:      input.Amount,
		Date:        date,
		Competence:  input.Competence,
		Type:        input.Type,
		Recurrent:   input.Recurrent,
	}
	if err := uc.repo.Create(ctx, income); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "income.create completed",
		slog.String("operation", "income.create"),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.String("income_id", income.ID.String()),
	)
	out := toOutput(income)
	return &out, nil
}
