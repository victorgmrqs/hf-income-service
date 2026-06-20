package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/victorgmrqs/hf-income-service/src/config"
	"github.com/victorgmrqs/hf-income-service/src/internal/handler"
	"github.com/victorgmrqs/hf-income-service/src/pkg/database"
	"github.com/victorgmrqs/hf-income-service/src/pkg/observability"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	logger := observability.NewLogger(cfg.Server.Env)
	metrics := observability.NewServiceMetrics("hf_income")

	if _, shutdown, err := observability.NewTracerProvider(context.Background()); err != nil {
		logger.Error("failed to initialize tracer provider", slog.String("error", err.Error()))
	} else {
		defer shutdown()
	}

	// Conexão validada na subida. O AutoMigrate das entidades entra no HF-37.
	if _, err := database.Connect(&cfg.Database); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	logger.Info("database connected")

	// Métricas Prometheus num servidor separado (scrape pelo Alloy em :APP_METRICS_PORT/metrics).
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		addr := ":" + cfg.Server.MetricsPort
		logger.Info("metrics server listening", slog.String("addr", addr))
		if err := http.ListenAndServe(addr, mux); err != nil {
			logger.Error("metrics server stopped", slog.String("error", err.Error()))
		}
	}()

	router := handler.SetupRouter(logger, metrics)
	logger.Info("server listening", slog.String("port", cfg.Server.Port))
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
