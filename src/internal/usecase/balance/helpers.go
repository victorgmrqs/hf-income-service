package balance

import (
	"context"
	"log/slog"
	"regexp"

	"go.opentelemetry.io/otel/trace"
)

// competenceRe valida o formato YYYY-MM com mês entre 01 e 12 (SAL-01).
var competenceRe = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

func isValidCompetence(competence string) bool {
	return competenceRe.MatchString(competence)
}

// traceIDFromContext extrai o trace_id do span ativo (vazio se não houver),
// para correlacionar logs com as traces no Grafana.
func traceIDFromContext(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.HasTraceID() {
		return ""
	}
	return sc.TraceID().String()
}

// logRuleViolation registra um WARN padronizado para violação de regra de negócio,
// com trace_id e rule_id (convenção de observabilidade do serviço).
func logRuleViolation(ctx context.Context, logger *slog.Logger, ruleID, reason string) {
	logger.WarnContext(ctx, "business rule violation",
		slog.String("trace_id", traceIDFromContext(ctx)),
		slog.String("rule_id", ruleID),
		slog.String("reason", reason),
	)
}
