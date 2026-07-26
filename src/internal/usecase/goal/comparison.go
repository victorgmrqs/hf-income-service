package goal

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
	"github.com/victorgmrqs/hf-income-service/src/pkg/httpclient"
)

type ComparisonUseCase interface {
	Execute(ctx context.Context, input ComparisonInput) ([]ComparisonItemOutput, error)
}

type comparisonUseCase struct {
	repo   repository.ReductionGoalRepository
	client httpclient.TransactionClient
	logger *slog.Logger
}

func NewComparisonUseCase(
	repo repository.ReductionGoalRepository,
	client httpclient.TransactionClient,
	logger *slog.Logger,
) ComparisonUseCase {
	return &comparisonUseCase{repo: repo, client: client, logger: logger}
}

// Execute monta o comparativo mensal (MET-05/07, FDD-003 §4): para cada meta da
// competência busca em paralelo o gasto atual da categoria no
// hf-transaction-service e calcula a variação vs. mês anterior, on_track e o
// progresso da meta. Falha de upstream degrada os campos daquele item para null
// (log WARN) — nunca vira erro da operação.
func (uc *comparisonUseCase) Execute(ctx context.Context, input ComparisonInput) ([]ComparisonItemOutput, error) {
	start := time.Now()
	uc.logger.InfoContext(ctx, "goal.comparison started",
		slog.String("operation", "goal.comparison"),
		slog.String("user_id", input.UserID.String()),
		slog.String("competence", input.Competence),
	)

	// MET-04: campos obrigatórios.
	if input.UserID == uuid.Nil || input.Competence == "" {
		logRuleViolation(ctx, uc.logger, "MET-04", "missing required field")
		return nil, ErrMissingRequiredField
	}
	// MET-04: competence YYYY-MM.
	if !isValidCompetence(input.Competence) {
		logRuleViolation(ctx, uc.logger, "MET-04", "invalid competence format")
		return nil, ErrInvalidCompetence
	}

	goals, err := uc.repo.ListByUserAndCompetence(ctx, input.UserID, input.Competence)
	if err != nil {
		return nil, err
	}

	items := make([]ComparisonItemOutput, len(goals))
	var wg sync.WaitGroup
	for i := range goals {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			items[i] = uc.buildItem(ctx, input.Competence, &goals[i])
		}(i)
	}
	wg.Wait()

	uc.logger.InfoContext(ctx, "goal.comparison completed",
		slog.String("operation", "goal.comparison"),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.Int("categories_count", len(items)),
	)
	return items, nil
}

// buildItem calcula a linha do comparativo de uma meta. CategoryName permanece
// null: o endpoint consumido do hf-transaction-service retorna apenas
// category_id e total — o frontend resolve o nome pelo id.
func (uc *comparisonUseCase) buildItem(ctx context.Context, competence string, goal *entity.ReductionGoal) ComparisonItemOutput {
	item := ComparisonItemOutput{
		CategoryID:          goal.CategoryID,
		PreviousMonthAmount: goal.PreviousAmount,
		TargetAmount:        goal.TargetAmount,
	}

	item.CurrentMonthAmount = uc.fetchCategoryTotal(ctx, goal, competence)

	// FDD-003 §4 (passo 5): snapshot ausente é preenchido retroativamente.
	if goal.PreviousAmount == nil {
		item.PreviousMonthAmount = uc.backfillPreviousAmount(ctx, goal)
	}

	current := item.CurrentMonthAmount
	if current == nil {
		return item
	}

	// MET-05/07: on_track e progresso da meta.
	onTrack := current.LessThanOrEqual(goal.TargetAmount)
	item.OnTrack = &onTrack
	if goal.TargetAmount.IsPositive() {
		progress := roundPct(current.Div(goal.TargetAmount).Mul(decimal.NewFromInt(100)))
		item.TargetProgressPct = &progress
	}
	// MET-05: variação percentual vs. mês anterior.
	if prev := item.PreviousMonthAmount; prev != nil && prev.IsPositive() {
		pct := roundPct(current.Sub(*prev).Div(*prev).Mul(decimal.NewFromInt(100)))
		label := variationLabel(pct)
		item.VariationPct = &pct
		item.VariationLabel = &label
	}
	return item
}

// fetchCategoryTotal busca o gasto atual da categoria; falha degrada para nil
// com log WARN (FDD-003 §7).
func (uc *comparisonUseCase) fetchCategoryTotal(ctx context.Context, goal *entity.ReductionGoal, competence string) *decimal.Decimal {
	result, err := uc.client.GetExpensesByCategory(ctx, goal.UserID.String(), goal.CategoryID.String(), competence)
	if err != nil || result == nil {
		reason := "upstream returned no data"
		if err != nil {
			reason = err.Error()
		}
		uc.logger.WarnContext(ctx, "goal.comparison current_month_amount unavailable",
			slog.String("trace_id", traceIDFromContext(ctx)),
			slog.String("user_id", goal.UserID.String()),
			slog.String("goal_id", goal.ID.String()),
			slog.String("category_id", goal.CategoryID.String()),
			slog.String("competence", competence),
			slog.String("reason", reason),
		)
		return nil
	}
	return &result.Total
}

// backfillPreviousAmount tenta preencher o snapshot do mês anterior quando ele
// não existe e o persiste para as próximas consultas; qualquer falha degrada
// para nil sem interromper o comparativo.
func (uc *comparisonUseCase) backfillPreviousAmount(ctx context.Context, goal *entity.ReductionGoal) *decimal.Decimal {
	prevCompetence, err := previousCompetence(goal.Competence)
	if err != nil {
		return nil
	}
	result, err := uc.client.GetExpensesByCategory(ctx, goal.UserID.String(), goal.CategoryID.String(), prevCompetence)
	if err != nil || result == nil {
		return nil
	}
	if err := uc.repo.UpdatePreviousAmount(ctx, goal.ID, result.Total); err != nil {
		uc.logger.WarnContext(ctx, "goal.comparison previous_amount not persisted",
			slog.String("trace_id", traceIDFromContext(ctx)),
			slog.String("goal_id", goal.ID.String()),
			slog.String("reason", err.Error()),
		)
	}
	return &result.Total
}

// roundPct converte um decimal em percentual com 1 casa (contrato FDD-003: -22.5, 88.6).
func roundPct(d decimal.Decimal) float64 {
	f, _ := d.Round(1).Float64()
	return f
}

// variationLabel monta o rótulo pt-BR da variação (MET-05),
// ex: "22,5% menor que o mês passado".
func variationLabel(pct float64) string {
	switch {
	case pct < 0:
		return formatPctBR(-pct) + "% menor que o mês passado"
	case pct > 0:
		return formatPctBR(pct) + "% maior que o mês passado"
	default:
		return "igual ao mês passado"
	}
}

func formatPctBR(pct float64) string {
	return strings.ReplaceAll(strconv.FormatFloat(pct, 'f', -1, 64), ".", ",")
}
