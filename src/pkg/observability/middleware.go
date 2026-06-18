package observability

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// RequestMiddleware instruments every HTTP request with:
//   - Distributed trace span (OTEL)
//   - Structured log with trace_id, method, route, status_code, duration_ms
//   - Prometheus metrics (latency + request count)
//
// Never logs request/response bodies — only metadata.
func RequestMiddleware(logger *slog.Logger, metrics *ServiceMetrics) gin.HandlerFunc {
	tracer := otel.Tracer("hf-income-service")

	return func(c *gin.Context) {
		ctx := otel.GetTextMapPropagator().Extract(
			c.Request.Context(),
			propagation.HeaderCarrier(c.Request.Header),
		)

		route := c.FullPath()
		if route == "" {
			route = "unknown"
		}

		ctx, span := tracer.Start(ctx, c.Request.Method+" "+route)
		defer span.End()

		c.Request = c.Request.WithContext(ctx)
		traceID := span.SpanContext().TraceID().String()
		start := time.Now()

		logger.InfoContext(ctx, "request started",
			slog.String("trace_id", traceID),
			slog.String("method", c.Request.Method),
			slog.String("route", route),
		)

		c.Next()

		duration := time.Since(start)
		statusCode := c.Writer.Status()
		statusStr := strconv.Itoa(statusCode)

		span.SetAttributes(
			semconv.HTTPStatusCode(statusCode),
			attribute.String("http.route", route),
		)

		logger.InfoContext(ctx, "request completed",
			slog.String("trace_id", traceID),
			slog.String("method", c.Request.Method),
			slog.String("route", route),
			slog.Int("status_code", statusCode),
			slog.Int64("duration_ms", duration.Milliseconds()),
		)

		metrics.HTTPRequestDuration.
			WithLabelValues(c.Request.Method, route, statusStr).
			Observe(duration.Seconds())
		metrics.HTTPRequestsTotal.
			WithLabelValues(c.Request.Method, route, statusStr).
			Inc()
	}
}
