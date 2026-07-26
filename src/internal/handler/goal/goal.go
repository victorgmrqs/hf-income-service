// Package goal contém os handlers HTTP do domínio de metas de redução (MET).
package goal

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	goalUseCase "github.com/victorgmrqs/hf-income-service/src/internal/usecase/goal"
	"github.com/victorgmrqs/hf-income-service/src/pkg/observability"
	"github.com/victorgmrqs/hf-income-service/src/pkg/response"
)

type GoalHandler struct {
	createUC     goalUseCase.CreateUseCase
	listUC       goalUseCase.ListUseCase
	updateUC     goalUseCase.UpdateUseCase
	deleteUC     goalUseCase.DeleteUseCase
	comparisonUC goalUseCase.ComparisonUseCase
	closeMonthUC goalUseCase.CloseMonthUseCase
	metrics      *observability.ServiceMetrics
}

func NewGoalHandler(
	createUC goalUseCase.CreateUseCase,
	listUC goalUseCase.ListUseCase,
	updateUC goalUseCase.UpdateUseCase,
	deleteUC goalUseCase.DeleteUseCase,
	comparisonUC goalUseCase.ComparisonUseCase,
	closeMonthUC goalUseCase.CloseMonthUseCase,
	metrics *observability.ServiceMetrics,
) *GoalHandler {
	return &GoalHandler{
		createUC:     createUC,
		listUC:       listUC,
		updateUC:     updateUC,
		deleteUC:     deleteUC,
		comparisonUC: comparisonUC,
		closeMonthUC: closeMonthUC,
		metrics:      metrics,
	}
}

func (h *GoalHandler) Create(c *gin.Context) {
	var body createGoalRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_USER_ID", "user_id must be a valid UUID")
		return
	}
	categoryID, err := uuid.Parse(body.CategoryID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_CATEGORY_ID", "category_id must be a valid UUID")
		return
	}

	out, err := h.createUC.Execute(c.Request.Context(), goalUseCase.CreateInput{
		UserID:       userID,
		CategoryID:   categoryID,
		Competence:   body.Competence,
		TargetAmount: body.TargetAmount,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, out)
}

func (h *GoalHandler) List(c *gin.Context) {
	userID, err := uuid.Parse(c.Query("user_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_USER_ID", "user_id query param is required and must be a valid UUID")
		return
	}
	out, err := h.listUC.Execute(c.Request.Context(), goalUseCase.ListInput{
		UserID:     userID,
		Competence: c.Query("competence"),
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, out)
}

func (h *GoalHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid reduction goal id")
		return
	}
	var body updateGoalRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	out, err := h.updateUC.Execute(c.Request.Context(), goalUseCase.UpdateInput{
		ID:           id,
		TargetAmount: body.TargetAmount,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, out)
}

func (h *GoalHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid reduction goal id")
		return
	}
	if err := h.deleteUC.Execute(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Comparison monta o card do comparativo mensal (MET-05/07) — responde 200
// mesmo com degradação parcial de upstream (campos null por item).
func (h *GoalHandler) Comparison(c *gin.Context) {
	userID, err := uuid.Parse(c.Query("user_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_USER_ID", "user_id query param is required and must be a valid UUID")
		return
	}
	out, err := h.comparisonUC.Execute(c.Request.Context(), goalUseCase.ComparisonInput{
		UserID:     userID,
		Competence: c.Query("competence"),
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, out)
}

// CloseMonth fecha a competência gravando achieved por meta (MET-06) —
// chamado como job; metas com upstream indisponível são puladas.
func (h *GoalHandler) CloseMonth(c *gin.Context) {
	var body closeMonthRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	out, err := h.closeMonthUC.Execute(c.Request.Context(), goalUseCase.CloseMonthInput{
		Competence: body.Competence,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, out)
}

// handleError mapeia os erros de domínio para o envelope/HTTP (FDD-003 §6) e
// incrementa BusinessErrorsTotal nas violações de regra (MET-04).
func (h *GoalHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, goalUseCase.ErrMissingRequiredField):
		h.metrics.BusinessErrorsTotal.WithLabelValues("MET", "MET-04").Inc()
		response.Error(c, http.StatusBadRequest, "MISSING_REQUIRED_FIELD", err.Error())
	case errors.Is(err, goalUseCase.ErrInvalidTargetAmount):
		h.metrics.BusinessErrorsTotal.WithLabelValues("MET", "MET-04").Inc()
		response.Error(c, http.StatusBadRequest, "INVALID_TARGET_AMOUNT", err.Error())
	case errors.Is(err, goalUseCase.ErrInvalidCompetence):
		h.metrics.BusinessErrorsTotal.WithLabelValues("MET", "MET-04").Inc()
		response.Error(c, http.StatusBadRequest, "INVALID_COMPETENCE", err.Error())
	case errors.Is(err, goalUseCase.ErrGoalAlreadyExists):
		h.metrics.BusinessErrorsTotal.WithLabelValues("MET", "MET-04").Inc()
		response.Error(c, http.StatusConflict, "GOAL_ALREADY_EXISTS", err.Error())
	case errors.Is(err, goalUseCase.ErrGoalNotFound):
		response.Error(c, http.StatusNotFound, "GOAL_NOT_FOUND", err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
	}
}

type createGoalRequest struct {
	UserID       string          `json:"user_id" binding:"required"`
	CategoryID   string          `json:"category_id" binding:"required"`
	Competence   string          `json:"competence" binding:"required"`
	TargetAmount decimal.Decimal `json:"target_amount" binding:"required"`
}

type updateGoalRequest struct {
	TargetAmount decimal.Decimal `json:"target_amount" binding:"required"`
}

type closeMonthRequest struct {
	Competence string `json:"competence" binding:"required"`
}
