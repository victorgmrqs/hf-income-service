// Package global_budget contém os handlers HTTP do domínio de teto global (ORC).
package global_budget

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	budgetUseCase "github.com/victorgmrqs/hf-income-service/src/internal/usecase/global_budget"
	"github.com/victorgmrqs/hf-income-service/src/pkg/observability"
	"github.com/victorgmrqs/hf-income-service/src/pkg/response"
)

type GlobalBudgetHandler struct {
	createUC     budgetUseCase.CreateUseCase
	getUC        budgetUseCase.GetUseCase
	updateUC     budgetUseCase.UpdateUseCase
	autoAdjustUC budgetUseCase.AutoAdjustUseCase
	previewUC    budgetUseCase.PreviewNextUseCase
	metrics      *observability.ServiceMetrics
}

func NewGlobalBudgetHandler(
	createUC budgetUseCase.CreateUseCase,
	getUC budgetUseCase.GetUseCase,
	updateUC budgetUseCase.UpdateUseCase,
	autoAdjustUC budgetUseCase.AutoAdjustUseCase,
	previewUC budgetUseCase.PreviewNextUseCase,
	metrics *observability.ServiceMetrics,
) *GlobalBudgetHandler {
	return &GlobalBudgetHandler{
		createUC:     createUC,
		getUC:        getUC,
		updateUC:     updateUC,
		autoAdjustUC: autoAdjustUC,
		previewUC:    previewUC,
		metrics:      metrics,
	}
}

func (h *GlobalBudgetHandler) Create(c *gin.Context) {
	var body createBudgetRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_USER_ID", "user_id must be a valid UUID")
		return
	}

	out, err := h.createUC.Execute(c.Request.Context(), budgetUseCase.CreateInput{
		UserID:     userID,
		Competence: body.Competence,
		Ceiling:    body.Ceiling,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, out)
}

func (h *GlobalBudgetHandler) Get(c *gin.Context) {
	userID, err := uuid.Parse(c.Query("user_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_USER_ID", "user_id query param is required and must be a valid UUID")
		return
	}
	out, err := h.getUC.Execute(c.Request.Context(), budgetUseCase.GetInput{
		UserID:     userID,
		Competence: c.Query("competence"),
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, out)
}

func (h *GlobalBudgetHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid global budget id")
		return
	}
	var body updateBudgetRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	out, err := h.updateUC.Execute(c.Request.Context(), budgetUseCase.UpdateInput{
		ID:      id,
		Ceiling: body.Ceiling,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, out)
}

// AutoAdjust aplica o auto-ajuste do teto para a competência de destino
// (ORC-03/04/05) — chamado pelo job de 1º do mês; idempotente.
func (h *GlobalBudgetHandler) AutoAdjust(c *gin.Context) {
	var body autoAdjustRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_USER_ID", "user_id must be a valid UUID")
		return
	}

	out, err := h.autoAdjustUC.Execute(c.Request.Context(), budgetUseCase.AutoAdjustInput{
		UserID:     userID,
		Competence: body.Competence,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, out)
}

// PreviewNext calcula o teto sugerido para a próxima competência sem persistir
// (ORC-03/04).
func (h *GlobalBudgetHandler) PreviewNext(c *gin.Context) {
	userID, err := uuid.Parse(c.Query("user_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_USER_ID", "user_id query param is required and must be a valid UUID")
		return
	}
	out, err := h.previewUC.Execute(c.Request.Context(), budgetUseCase.PreviewInput{UserID: userID})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, out)
}

// handleError mapeia os erros de domínio para o envelope/HTTP e incrementa
// BusinessErrorsTotal nas violações de regra (com o respectivo rule_id).
func (h *GlobalBudgetHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, budgetUseCase.ErrMissingRequiredField):
		h.metrics.BusinessErrorsTotal.WithLabelValues("ORC", "ORC-01").Inc()
		response.Error(c, http.StatusBadRequest, "MISSING_REQUIRED_FIELD", err.Error())
	case errors.Is(err, budgetUseCase.ErrInvalidCeiling):
		h.metrics.BusinessErrorsTotal.WithLabelValues("ORC", "ORC-01").Inc()
		response.Error(c, http.StatusBadRequest, "INVALID_CEILING", err.Error())
	case errors.Is(err, budgetUseCase.ErrInvalidCompetence):
		h.metrics.BusinessErrorsTotal.WithLabelValues("ORC", "ORC-01").Inc()
		response.Error(c, http.StatusBadRequest, "INVALID_COMPETENCE", err.Error())
	case errors.Is(err, budgetUseCase.ErrBudgetAlreadyExists):
		h.metrics.BusinessErrorsTotal.WithLabelValues("ORC", "ORC-01").Inc()
		response.Error(c, http.StatusConflict, "BUDGET_ALREADY_EXISTS", err.Error())
	case errors.Is(err, budgetUseCase.ErrBudgetNotFound):
		response.Error(c, http.StatusNotFound, "BUDGET_NOT_FOUND", err.Error())
	case errors.Is(err, budgetUseCase.ErrNoPreviousBudget):
		h.metrics.BusinessErrorsTotal.WithLabelValues("ORC", "ORC-03").Inc()
		response.Error(c, http.StatusBadRequest, "NO_PREVIOUS_BUDGET", err.Error())
	case errors.Is(err, budgetUseCase.ErrUpstreamUnavailable):
		h.metrics.BusinessErrorsTotal.WithLabelValues("ORC", "upstream").Inc()
		response.Error(c, http.StatusServiceUnavailable, "UPSTREAM_UNAVAILABLE", "could not fetch expense data, please retry")
	default:
		response.Error(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
	}
}

type autoAdjustRequest struct {
	UserID     string `json:"user_id" binding:"required"`
	Competence string `json:"competence" binding:"required"`
}

type createBudgetRequest struct {
	UserID     string          `json:"user_id" binding:"required"`
	Competence string          `json:"competence" binding:"required"`
	Ceiling    decimal.Decimal `json:"ceiling" binding:"required"`
}

type updateBudgetRequest struct {
	Ceiling decimal.Decimal `json:"ceiling" binding:"required"`
}
