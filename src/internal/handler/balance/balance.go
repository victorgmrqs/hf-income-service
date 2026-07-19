// Package balance contém o handler HTTP do domínio de saldo mensal (SAL).
package balance

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	balanceUseCase "github.com/victorgmrqs/hf-income-service/src/internal/usecase/balance"
	"github.com/victorgmrqs/hf-income-service/src/pkg/observability"
	"github.com/victorgmrqs/hf-income-service/src/pkg/response"
)

type BalanceHandler struct {
	getUC   balanceUseCase.GetUseCase
	metrics *observability.ServiceMetrics
}

func NewBalanceHandler(getUC balanceUseCase.GetUseCase, metrics *observability.ServiceMetrics) *BalanceHandler {
	return &BalanceHandler{getUC: getUC, metrics: metrics}
}

// Get calcula o saldo mensal do usuário (SAL-01..05, ORC-06/07) — endpoint
// central do Dashboard; sem persistência.
func (h *BalanceHandler) Get(c *gin.Context) {
	userID, err := uuid.Parse(c.Query("user_id"))
	if err != nil {
		// SAL-01: user_id ausente/inválido.
		h.metrics.BusinessErrorsTotal.WithLabelValues("SAL", "SAL-01").Inc()
		response.Error(c, http.StatusBadRequest, "MISSING_REQUIRED_FIELD", "user_id query param is required and must be a valid UUID")
		return
	}

	out, err := h.getUC.Execute(c.Request.Context(), balanceUseCase.GetInput{
		UserID:     userID,
		Competence: c.Query("competence"),
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, out)
}

// handleError mapeia os erros de domínio para o envelope/HTTP e incrementa
// BusinessErrorsTotal (matriz de erros do FDD-004 §6).
func (h *BalanceHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, balanceUseCase.ErrMissingRequiredField):
		h.metrics.BusinessErrorsTotal.WithLabelValues("SAL", "SAL-01").Inc()
		response.Error(c, http.StatusBadRequest, "MISSING_REQUIRED_FIELD", err.Error())
	case errors.Is(err, balanceUseCase.ErrInvalidCompetence):
		h.metrics.BusinessErrorsTotal.WithLabelValues("SAL", "SAL-01").Inc()
		response.Error(c, http.StatusBadRequest, "INVALID_COMPETENCE", err.Error())
	case errors.Is(err, balanceUseCase.ErrUpstreamTimeout):
		h.metrics.BusinessErrorsTotal.WithLabelValues("SAL", "upstream").Inc()
		response.Error(c, http.StatusServiceUnavailable, "UPSTREAM_TIMEOUT", err.Error())
	case errors.Is(err, balanceUseCase.ErrUpstreamError):
		h.metrics.BusinessErrorsTotal.WithLabelValues("SAL", "upstream").Inc()
		response.Error(c, http.StatusBadGateway, "UPSTREAM_ERROR", err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
	}
}
