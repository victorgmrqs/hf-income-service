package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	balancehandler "github.com/victorgmrqs/hf-income-service/src/internal/handler/balance"
	globalbudgethandler "github.com/victorgmrqs/hf-income-service/src/internal/handler/global_budget"
	goalhandler "github.com/victorgmrqs/hf-income-service/src/internal/handler/goal"
	incomehandler "github.com/victorgmrqs/hf-income-service/src/internal/handler/income"
	"github.com/victorgmrqs/hf-income-service/src/pkg/observability"
)

// SetupRouter monta o roteador Gin do serviço: Recovery, middleware de
// observabilidade (trace + log + métricas), health check e os grupos de rota
// de domínio sob /api/v1.
func SetupRouter(
	logger *slog.Logger,
	metrics *observability.ServiceMetrics,
	incomeHandler *incomehandler.IncomeHandler,
	globalBudgetHandler *globalbudgethandler.GlobalBudgetHandler,
	balanceHandler *balancehandler.BalanceHandler,
	goalHandler *goalhandler.GoalHandler,
) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(observability.RequestMiddleware(logger, metrics))

	// Liveness probe — resposta simples (não usa o envelope {data,error}).
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := router.Group("/api/v1")
	{
		income := api.Group("/income")
		{
			income.POST("", incomeHandler.Create)
			income.POST("/propagate", incomeHandler.Propagate)
			income.GET("", incomeHandler.List)
			income.GET("/:id", incomeHandler.GetByID)
			income.PUT("/:id", incomeHandler.Update)
			income.DELETE("/:id", incomeHandler.Delete)
		}

		budgets := api.Group("/budgets/global")
		{
			budgets.POST("", globalBudgetHandler.Create)
			budgets.GET("", globalBudgetHandler.Get)
			// Rotas literais antes de /:id para não conflitar (ORC-03/04/05).
			budgets.GET("/preview-next", globalBudgetHandler.PreviewNext)
			budgets.POST("/auto-adjust", globalBudgetHandler.AutoAdjust)
			budgets.PUT("/:id", globalBudgetHandler.Update)
		}

		// Metas de redução (MET) — FDD-003 §5.
		goals := api.Group("/goals/reduction")
		{
			goals.POST("", goalHandler.Create)
			goals.GET("", goalHandler.List)
			// Rotas literais do domínio (MET-05/06/07).
			goals.GET("/comparison", goalHandler.Comparison)
			goals.POST("/close-month", goalHandler.CloseMonth)
			goals.PUT("/:id", goalHandler.Update)
			goals.DELETE("/:id", goalHandler.Delete)
		}

		// Saldo mensal (SAL) — calculado sob demanda, sem persistência.
		api.GET("/balance", balanceHandler.Get)
	}

	return router
}
