package goal

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
	"github.com/victorgmrqs/hf-income-service/src/pkg/httpclient"
)

type CreateUseCase interface {
	Execute(ctx context.Context, input CreateInput) (*GoalOutput, error)
}

type createUseCase struct {
	repo   repository.ReductionGoalRepository
	client httpclient.TransactionClient
	logger *slog.Logger
}

func NewCreateUseCase(
	repo repository.ReductionGoalRepository,
	client httpclient.TransactionClient,
	logger *slog.Logger,
) CreateUseCase {
	return &createUseCase{repo: repo, client: client, logger: logger}
}

// Execute cria uma meta de redução (MET-04, FDD-003 §4): valida os campos
// obrigatórios, garante unicidade (user_id, category_id, competence) e tenta
// capturar o snapshot de previous_amount via hf-transaction-service — falha
// upstream não bloqueia a criação, apenas degrada o snapshot para null.
func (uc *createUseCase) Execute(ctx context.Context, input CreateInput) (*GoalOutput, error) {
	start := time.Now()
	uc.logger.InfoContext(ctx, "goal.create started",
		slog.String("operation", "goal.create"),
		slog.String("user_id", input.UserID.String()),
		slog.String("competence", input.Competence),
	)

	// MET-04: campos obrigatórios.
	if input.UserID == uuid.Nil || input.CategoryID == uuid.Nil || input.Competence == "" {
		logRuleViolation(ctx, uc.logger, "MET-04", "missing required field")
		return nil, ErrMissingRequiredField
	}
	// MET-04: target_amount > 0.
	if !input.TargetAmount.IsPositive() {
		logRuleViolation(ctx, uc.logger, "MET-04", "target amount must be greater than zero")
		return nil, ErrInvalidTargetAmount
	}
	// MET-04: competence YYYY-MM.
	if !isValidCompetence(input.Competence) {
		logRuleViolation(ctx, uc.logger, "MET-04", "invalid competence format")
		return nil, ErrInvalidCompetence
	}

	// MET-04: uma meta por (user_id, category_id, competence).
	exists, err := uc.repo.ExistsByUserCategoryAndCompetence(ctx, input.UserID, input.CategoryID, input.Competence)
	if err != nil {
		return nil, err
	}
	if exists {
		logRuleViolation(ctx, uc.logger, "MET-04", "goal already exists for user, category and competence")
		return nil, ErrGoalAlreadyExists
	}

	previousAmount := uc.fetchPreviousAmount(ctx, input)

	goal := &entity.ReductionGoal{
		UserID:         input.UserID,
		CategoryID:     input.CategoryID,
		Competence:     input.Competence,
		TargetAmount:   input.TargetAmount,
		PreviousAmount: previousAmount,
		// Achieved permanece nil: mês em curso (MET-06 só fecha no job de fechamento).
	}
	if err := uc.repo.Create(ctx, goal); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "goal.create completed",
		slog.String("operation", "goal.create"),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.String("goal_id", goal.ID.String()),
	)
	out := toOutput(goal)
	return &out, nil
}

// fetchPreviousAmount busca o gasto da categoria na competência anterior para
// preencher o snapshot inicial. Qualquer falha (competência anterior inválida
// ou upstream indisponível) degrada para nil sem bloquear a criação (FDD-003 §4/§7).
func (uc *createUseCase) fetchPreviousAmount(ctx context.Context, input CreateInput) *decimal.Decimal {
	prevCompetence, err := previousCompetence(input.Competence)
	if err != nil {
		uc.logger.WarnContext(ctx, "goal.create previous_amount unavailable",
			slog.String("trace_id", traceIDFromContext(ctx)),
			slog.String("user_id", input.UserID.String()),
			slog.String("category_id", input.CategoryID.String()),
			slog.String("competence", input.Competence),
			slog.String("reason", err.Error()),
		)
		return nil
	}

	result, err := uc.client.GetExpensesByCategory(ctx, input.UserID.String(), input.CategoryID.String(), prevCompetence)
	if err != nil {
		uc.logger.WarnContext(ctx, "goal.create previous_amount unavailable",
			slog.String("trace_id", traceIDFromContext(ctx)),
			slog.String("user_id", input.UserID.String()),
			slog.String("category_id", input.CategoryID.String()),
			slog.String("competence", input.Competence),
			slog.String("reason", err.Error()),
		)
		return nil
	}
	if result == nil {
		return nil
	}
	return &result.Total
}
