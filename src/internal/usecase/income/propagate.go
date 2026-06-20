package income

import (
	"context"
	"log/slog"
	"time"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
)

type PropagateInput struct {
	Competence string // competência DESTINO (YYYY-MM)
}

type PropagateOutput struct {
	Propagated int `json:"propagated"`
}

type PropagateUseCase interface {
	Execute(ctx context.Context, input PropagateInput) (*PropagateOutput, error)
}

type propagateUseCase struct {
	repo   repository.IncomeRepository
	logger *slog.Logger
}

func NewPropagateUseCase(repo repository.IncomeRepository, logger *slog.Logger) PropagateUseCase {
	return &propagateUseCase{repo: repo, logger: logger}
}

// Execute copia as receitas recorrentes da competência anterior para a competência
// destino (REC-04). É idempotente: cópias já existentes (mesmo origin_id + competência)
// são ignoradas.
func (uc *propagateUseCase) Execute(ctx context.Context, input PropagateInput) (*PropagateOutput, error) {
	start := time.Now()
	uc.logger.InfoContext(ctx, "income.propagate started",
		slog.String("operation", "income.propagate"),
		slog.String("competence", input.Competence),
	)

	if !isValidCompetence(input.Competence) {
		logRuleViolation(ctx, uc.logger, "REC-01", "invalid competence format")
		return nil, ErrInvalidCompetence
	}

	previous, err := previousCompetence(input.Competence)
	if err != nil {
		logRuleViolation(ctx, uc.logger, "REC-01", "invalid competence format")
		return nil, ErrInvalidCompetence
	}

	sources, err := uc.repo.ListRecurrentByCompetence(ctx, previous)
	if err != nil {
		return nil, err
	}

	propagated := 0
	for i := range sources {
		src := sources[i]

		// origin_id sempre aponta para a receita raiz (REC-03/REC-04).
		originID := src.ID
		if src.OriginID != nil {
			originID = *src.OriginID
		}

		exists, err := uc.repo.ExistsByOriginAndCompetence(ctx, originID, input.Competence)
		if err != nil {
			return nil, err
		}
		if exists {
			continue // idempotência
		}

		copyID := originID
		cp := &entity.Income{
			UserID:      src.UserID,
			Description: src.Description,
			Amount:      src.Amount,
			Date:        dateInCompetence(src.Date, input.Competence),
			Competence:  input.Competence,
			Type:        src.Type,
			Recurrent:   true,
			OriginID:    &copyID,
		}
		if err := uc.repo.Create(ctx, cp); err != nil {
			return nil, err
		}
		propagated++
	}

	uc.logger.InfoContext(ctx, "income.propagate completed",
		slog.String("operation", "income.propagate"),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.Int("propagated", propagated),
	)
	return &PropagateOutput{Propagated: propagated}, nil
}

// previousCompetence retorna a competência imediatamente anterior (YYYY-MM).
func previousCompetence(competence string) (string, error) {
	t, err := time.Parse("2006-01", competence)
	if err != nil {
		return "", err
	}
	return t.AddDate(0, -1, 0).Format("2006-01"), nil
}

// dateInCompetence mantém o dia-do-mês da data original na competência destino,
// truncando para o último dia quando o mês destino é mais curto.
func dateInCompetence(original time.Time, competence string) time.Time {
	ym, err := time.Parse("2006-01", competence)
	if err != nil {
		return original
	}
	day := original.Day()
	lastDay := time.Date(ym.Year(), ym.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(ym.Year(), ym.Month(), day, 0, 0, 0, 0, time.UTC)
}
