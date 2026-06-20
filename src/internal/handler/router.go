package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

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
			income.GET("", incomeHandler.List)
			income.GET("/:id", incomeHandler.GetByID)
			income.PUT("/:id", incomeHandler.Update)
			income.DELETE("/:id", incomeHandler.Delete)
		}
	}

	return router
}
