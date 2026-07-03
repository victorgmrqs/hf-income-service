package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	globalbudgethandler "github.com/victorgmrqs/hf-income-service/src/internal/handler/global_budget"
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
) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(CORSMiddleware())
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
			budgets.PUT("/:id", globalBudgetHandler.Update)
		}

		// Rota temporária de mock para o Saldo (SAL), que será implementado na Fase 4 (T17/T18)
		api.GET("/balance", func(c *gin.Context) {
			competence := c.DefaultQuery("competence", "2026-07")
			c.JSON(http.StatusOK, gin.H{
				"data": gin.H{
					"competence":                  competence,
					"total_income":                7500.00,
					"total_expenses":              3200.00,
					"balance_today":               4300.00,
					"committed_bills":             850.00,
					"projected_balance":           3450.00,
					"global_ceiling":              5000.00,
					"ceiling_usage_pct":           64.0,
					"projected_ceiling_usage_pct": 81.0,
					"is_projected_negative":       false,
				},
			})
		})
	}

	return router
}
