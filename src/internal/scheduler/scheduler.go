// Package scheduler dispara os jobs mensais do serviço no 1º dia do mês
// (HF-38): propagação de receitas recorrentes (REC-04) e auto-ajuste do teto
// global por usuário (ORC-05). É um driver de use cases — como o handler HTTP,
// não contém lógica de negócio própria; a idempotência vem dos próprios use
// cases (índice único da propagação e upsert do auto-ajuste), então re-execuções
// após restart são seguras.
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
	budgetUseCase "github.com/victorgmrqs/hf-income-service/src/internal/usecase/global_budget"
	incomeUseCase "github.com/victorgmrqs/hf-income-service/src/internal/usecase/income"
)

const competenceLayout = "2006-01"

type Scheduler struct {
	propagateUC  incomeUseCase.PropagateUseCase
	autoAdjustUC budgetUseCase.AutoAdjustUseCase
	budgetRepo   repository.GlobalBudgetRepository
	logger       *slog.Logger
	enabled      bool
	loc          *time.Location

	// Injetáveis nos testes.
	nowFn    func() time.Time
	interval time.Duration

	// Dedupe intra-processo: competência já executada neste processo não repete
	// a cada tick do dia 1. Reprocesso manual fica nos endpoints (idempotentes).
	lastCompetence string
}

// New monta o scheduler. TZ inválida não derruba o boot: loga WARN e usa UTC.
func New(
	propagateUC incomeUseCase.PropagateUseCase,
	autoAdjustUC budgetUseCase.AutoAdjustUseCase,
	budgetRepo repository.GlobalBudgetRepository,
	logger *slog.Logger,
	enabled bool,
	tz string,
) *Scheduler {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		logger.Warn("invalid SCHEDULER_TZ, falling back to UTC",
			slog.String("tz", tz),
			slog.String("error", err.Error()),
		)
		loc = time.UTC
	}
	return &Scheduler{
		propagateUC:  propagateUC,
		autoAdjustUC: autoAdjustUC,
		budgetRepo:   budgetRepo,
		logger:       logger,
		enabled:      enabled,
		loc:          loc,
		nowFn:        time.Now,
		interval:     time.Hour,
	}
}

// Run bloqueia até o contexto ser cancelado (rodar em goroutine). Checa
// imediatamente no boot — restart no dia 1 dispara o ciclo sem esperar o tick.
func (s *Scheduler) Run(ctx context.Context) {
	if !s.enabled {
		s.logger.Info("scheduler disabled (SCHEDULER_ENABLED=false)")
		return
	}
	s.logger.Info("scheduler started",
		slog.String("tz", s.loc.String()),
		slog.String("interval", s.interval.String()),
	)

	s.tick(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler stopped", slog.String("reason", ctx.Err().Error()))
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

// tick executa o ciclo quando é dia 1 na TZ configurada e a competência ainda
// não rodou neste processo (ORC-05/REC-04: job do 1º dia do mês).
func (s *Scheduler) tick(ctx context.Context) {
	now := s.nowFn().In(s.loc)
	if now.Day() != 1 {
		return
	}
	competence := now.Format(competenceLayout)
	if competence == s.lastCompetence {
		return
	}
	s.runCycle(ctx, competence)
	s.lastCompetence = competence
}

// runCycle dispara os dois jobs. Falha em um job/usuário loga ERROR e segue —
// o ciclo nunca derruba o servidor nem aborta os demais.
func (s *Scheduler) runCycle(ctx context.Context, competence string) {
	start := time.Now()
	s.logger.InfoContext(ctx, "scheduler.monthly_jobs started",
		slog.String("operation", "scheduler.monthly_jobs"),
		slog.String("competence", competence),
	)

	propagated, failed := 0, 0

	// REC-04: propaga receitas recorrentes para a competência corrente.
	if out, err := s.propagateUC.Execute(ctx, incomeUseCase.PropagateInput{Competence: competence}); err != nil {
		failed++
		s.logger.ErrorContext(ctx, "scheduler propagate failed",
			slog.String("trace_id", traceIDFromContext(ctx)),
			slog.String("competence", competence),
			slog.String("error", err.Error()),
		)
	} else {
		propagated = out.Propagated
	}

	// ORC-05: auto-ajuste por usuário com teto ativo na competência anterior
	// (base do cálculo ORC-03/04 — evita ErrNoPreviousBudget em massa).
	adjusted := 0
	prevCompetence, err := previousCompetence(competence)
	if err != nil {
		failed++
		s.logger.ErrorContext(ctx, "scheduler could not derive previous competence",
			slog.String("trace_id", traceIDFromContext(ctx)),
			slog.String("competence", competence),
			slog.String("error", err.Error()),
		)
	} else if userIDs, err := s.budgetRepo.ListUserIDsByCompetence(ctx, prevCompetence); err != nil {
		failed++
		s.logger.ErrorContext(ctx, "scheduler could not list users for auto-adjust",
			slog.String("trace_id", traceIDFromContext(ctx)),
			slog.String("competence", prevCompetence),
			slog.String("error", err.Error()),
		)
	} else {
		for _, userID := range userIDs {
			if ctx.Err() != nil {
				break
			}
			if _, err := s.autoAdjustUC.Execute(ctx, budgetUseCase.AutoAdjustInput{
				UserID:     userID,
				Competence: competence,
			}); err != nil {
				failed++
				s.logger.ErrorContext(ctx, "scheduler auto-adjust failed",
					slog.String("trace_id", traceIDFromContext(ctx)),
					slog.String("user_id", userID.String()),
					slog.String("competence", competence),
					slog.String("error", err.Error()),
				)
				continue
			}
			adjusted++
		}
	}

	s.logger.InfoContext(ctx, "scheduler.monthly_jobs completed",
		slog.String("operation", "scheduler.monthly_jobs"),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.String("competence", competence),
		slog.Int("propagated", propagated),
		slog.Int("adjusted", adjusted),
		slog.Int("failed", failed),
	)
}

// previousCompetence retorna a competência anterior (YYYY-MM), cobrindo virada de ano.
func previousCompetence(competence string) (string, error) {
	t, err := time.Parse(competenceLayout, competence)
	if err != nil {
		return "", err
	}
	return t.AddDate(0, -1, 0).Format(competenceLayout), nil
}

// traceIDFromContext extrai o trace_id do span ativo (vazio em execução de
// background sem trace) — convenção de observabilidade do serviço.
func traceIDFromContext(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.HasTraceID() {
		return ""
	}
	return sc.TraceID().String()
}
