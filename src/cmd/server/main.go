package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/victorgmrqs/hf-income-service/src/config"
	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
	"github.com/victorgmrqs/hf-income-service/src/internal/handler"
	balancehandler "github.com/victorgmrqs/hf-income-service/src/internal/handler/balance"
	globalbudgethandler "github.com/victorgmrqs/hf-income-service/src/internal/handler/global_budget"
	incomehandler "github.com/victorgmrqs/hf-income-service/src/internal/handler/income"
	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
	balanceUseCase "github.com/victorgmrqs/hf-income-service/src/internal/usecase/balance"
	budgetUseCase "github.com/victorgmrqs/hf-income-service/src/internal/usecase/global_budget"
	incomeUseCase "github.com/victorgmrqs/hf-income-service/src/internal/usecase/income"
	"github.com/victorgmrqs/hf-income-service/src/pkg/database"
	"github.com/victorgmrqs/hf-income-service/src/pkg/httpclient"
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

	db, err := database.Connect(&cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err := db.AutoMigrate(&entity.Income{}, &entity.GlobalBudget{}); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	// ORC-01 (HF-44): índice único parcial (user_id, competence) WHERE deleted_at IS NULL.
	if err := repository.EnsureGlobalBudgetIndexes(db); err != nil {
		log.Fatalf("failed to ensure global budget indexes: %v", err)
	}
	logger.Info("database connected and migrated")

	// Wiring REC: repository -> use cases -> handler.
	incomeRepo := repository.NewIncomeRepository(db)
	incomeHandler := incomehandler.NewIncomeHandler(
		incomeUseCase.NewCreateUseCase(incomeRepo, logger),
		incomeUseCase.NewGetUseCase(incomeRepo, logger),
		incomeUseCase.NewListUseCase(incomeRepo, logger),
		incomeUseCase.NewUpdateUseCase(incomeRepo, logger),
		incomeUseCase.NewDeleteUseCase(incomeRepo, logger),
		incomeUseCase.NewPropagateUseCase(incomeRepo, logger),
		metrics,
	)

	// Wiring ORC: repository + httpclient -> use cases -> handler.
	// O auto-ajuste (ORC-03/04) consome o hf-transaction-service via TransactionClient.
	txClient := httpclient.NewTransactionClient(cfg.Transaction.URL, nil)
	globalBudgetRepo := repository.NewGlobalBudgetRepository(db)
	globalBudgetHandler := globalbudgethandler.NewGlobalBudgetHandler(
		budgetUseCase.NewCreateUseCase(globalBudgetRepo, logger),
		budgetUseCase.NewGetUseCase(globalBudgetRepo, logger),
		budgetUseCase.NewUpdateUseCase(globalBudgetRepo, logger),
		budgetUseCase.NewAutoAdjustUseCase(globalBudgetRepo, txClient, logger),
		budgetUseCase.NewPreviewNextUseCase(globalBudgetRepo, txClient, logger),
		metrics,
	)

	// Wiring SAL: repositórios locais + httpclient -> use case -> handler.
	// Saldo é calculado sob demanda — não há repositório próprio.
	balanceHandler := balancehandler.NewBalanceHandler(
		balanceUseCase.NewGetUseCase(incomeRepo, globalBudgetRepo, txClient, logger),
		metrics,
	)

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

	router := handler.SetupRouter(logger, metrics, incomeHandler, globalBudgetHandler, balanceHandler)
	logger.Info("server listening", slog.String("port", cfg.Server.Port))
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
