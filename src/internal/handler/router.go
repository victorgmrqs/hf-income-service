package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/victorgmrqs/hf-income-service/src/pkg/observability"
)

// SetupRouter monta o roteador Gin do serviço: Recovery, middleware de
// observabilidade (trace + log + métricas) e o health check. Os grupos de rota
// de domínio (/api/v1/...) são adicionados pelas tasks de cada domínio.
func SetupRouter(logger *slog.Logger, metrics *observability.ServiceMetrics) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(observability.RequestMiddleware(logger, metrics))

	// Liveness probe — resposta simples (não usa o envelope {data,error}).
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return router
}
