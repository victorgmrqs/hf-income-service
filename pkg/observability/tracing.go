package observability

import (
	"context"
	"os"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// NewTracerProvider configures OpenTelemetry with OTLP HTTP exporter (Grafana Cloud / Jaeger).
//
// Required env vars:
//
//	OTEL_EXPORTER_OTLP_ENDPOINT  — e.g. https://otlp-gateway-prod-us-east-0.grafana.net/otlp
//	OTEL_EXPORTER_OTLP_HEADERS   — e.g. Authorization=Basic <base64(instanceId:token)>
//
// Optional env vars:
//
//	OTEL_SERVICE_NAME    — defaults to "hf-income-service"
//	OTEL_SAMPLING_RATIO  — float 0.0–1.0, defaults to 1.0 (100%)
//
// Returns the provider and a shutdown function to flush pending spans on exit.
func NewTracerProvider(ctx context.Context) (*sdktrace.TracerProvider, func(), error) {
	exporter, err := otlptracehttp.New(ctx) // reads OTEL_EXPORTER_OTLP_* from env automatically
	if err != nil {
		return nil, nil, err
	}

	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "hf-income-service"
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(parseSampler(os.Getenv("OTEL_SAMPLING_RATIO"))),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.DeploymentEnvironment(os.Getenv("APP_ENV")),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	shutdown := func() { _ = tp.Shutdown(context.Background()) }
	return tp, shutdown, nil
}

func parseSampler(ratio string) sdktrace.Sampler {
	if ratio == "" {
		return sdktrace.AlwaysSample()
	}
	r, err := strconv.ParseFloat(ratio, 64)
	if err != nil || r >= 1.0 {
		return sdktrace.AlwaysSample()
	}
	if r <= 0 {
		return sdktrace.NeverSample()
	}
	return sdktrace.TraceIDRatioBased(r)
}
