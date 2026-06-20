package income

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"
)

const dateLayout = "2006-01-02"

func parseDate(s string) (time.Time, error) {
	return time.Parse(dateLayout, s)
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
