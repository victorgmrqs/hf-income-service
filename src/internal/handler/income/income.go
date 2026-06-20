// Package income contém os handlers HTTP do domínio de receitas (REC).
package income

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
	incomeUseCase "github.com/victorgmrqs/hf-income-service/src/internal/usecase/income"
	"github.com/victorgmrqs/hf-income-service/src/pkg/observability"
	"github.com/victorgmrqs/hf-income-service/src/pkg/response"
)

type IncomeHandler struct {
	createUC    incomeUseCase.CreateUseCase
	getUC       incomeUseCase.GetUseCase
	listUC      incomeUseCase.ListUseCase
	updateUC    incomeUseCase.UpdateUseCase
	deleteUC    incomeUseCase.DeleteUseCase
	propagateUC incomeUseCase.PropagateUseCase
	metrics     *observability.ServiceMetrics
}

func NewIncomeHandler(
	createUC incomeUseCase.CreateUseCase,
	getUC incomeUseCase.GetUseCase,
	listUC incomeUseCase.ListUseCase,
	updateUC incomeUseCase.UpdateUseCase,
	deleteUC incomeUseCase.DeleteUseCase,
	propagateUC incomeUseCase.PropagateUseCase,
	metrics *observability.ServiceMetrics,
) *IncomeHandler {
	return &IncomeHandler{
		createUC:    createUC,
		getUC:       getUC,
		listUC:      listUC,
		updateUC:    updateUC,
		deleteUC:    deleteUC,
		propagateUC: propagateUC,
		metrics:     metrics,
	}
}

func (h *IncomeHandler) Create(c *gin.Context) {
	var body createIncomeRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_USER_ID", "user_id must be a valid UUID")
		return
	}

	out, err := h.createUC.Execute(c.Request.Context(), incomeUseCase.CreateInput{
		UserID:      userID,
		Description: body.Description,
		Amount:      body.Amount,
		Date:        body.Date,
		Competence:  body.Competence,
		Type:        entity.IncomeType(body.Type),
		Recurrent:   body.Recurrent,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, out)
}

func (h *IncomeHandler) List(c *gin.Context) {
	userID, err := uuid.Parse(c.Query("user_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_USER_ID", "user_id query param is required and must be a valid UUID")
		return
	}
	out, err := h.listUC.Execute(c.Request.Context(), incomeUseCase.ListInput{
		UserID:     userID,
		Competence: c.Query("competence"),
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, out)
}

func (h *IncomeHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid income id")
		return
	}
	out, err := h.getUC.Execute(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, out)
}

func (h *IncomeHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid income id")
		return
	}
	var body updateIncomeRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	input := incomeUseCase.UpdateInput{
		ID:          id,
		Description: body.Description,
		Amount:      body.Amount,
		Date:        body.Date,
		Recurrent:   body.Recurrent,
	}
	if body.Type != nil {
		t := entity.IncomeType(*body.Type)
		input.Type = &t
	}

	out, err := h.updateUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, out)
}

func (h *IncomeHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid income id")
		return
	}
	if err := h.deleteUC.Execute(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusNoContent, nil)
}

func (h *IncomeHandler) Propagate(c *gin.Context) {
	var body propagateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	out, err := h.propagateUC.Execute(c.Request.Context(), incomeUseCase.PropagateInput{
		Competence: body.Competence,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, out)
}

// handleError mapeia os erros de domínio para o envelope/HTTP e incrementa
// BusinessErrorsTotal nas violações de regra (com o respectivo rule_id).
func (h *IncomeHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, incomeUseCase.ErrMissingRequiredField):
		h.metrics.BusinessErrorsTotal.WithLabelValues("REC", "REC-01").Inc()
		response.Error(c, http.StatusBadRequest, "MISSING_REQUIRED_FIELD", err.Error())
	case errors.Is(err, incomeUseCase.ErrInvalidAmount):
		h.metrics.BusinessErrorsTotal.WithLabelValues("REC", "REC-02").Inc()
		response.Error(c, http.StatusBadRequest, "INVALID_AMOUNT", err.Error())
	case errors.Is(err, incomeUseCase.ErrInvalidIncomeType):
		h.metrics.BusinessErrorsTotal.WithLabelValues("REC", "REC-01").Inc()
		response.Error(c, http.StatusBadRequest, "INVALID_INCOME_TYPE", err.Error())
	case errors.Is(err, incomeUseCase.ErrInvalidCompetence):
		h.metrics.BusinessErrorsTotal.WithLabelValues("REC", "REC-01").Inc()
		response.Error(c, http.StatusBadRequest, "INVALID_COMPETENCE", err.Error())
	case errors.Is(err, incomeUseCase.ErrInvalidDate):
		h.metrics.BusinessErrorsTotal.WithLabelValues("REC", "REC-01").Inc()
		response.Error(c, http.StatusBadRequest, "INVALID_DATE", err.Error())
	case errors.Is(err, incomeUseCase.ErrCannotEditPropagatedIncome):
		h.metrics.BusinessErrorsTotal.WithLabelValues("REC", "REC-03").Inc()
		response.Error(c, http.StatusBadRequest, "CANNOT_EDIT_PROPAGATED_INCOME", err.Error())
	case errors.Is(err, incomeUseCase.ErrIncomeNotFound):
		response.Error(c, http.StatusNotFound, "INCOME_NOT_FOUND", err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
	}
}

type createIncomeRequest struct {
	UserID      string          `json:"user_id" binding:"required"`
	Description string          `json:"description" binding:"required"`
	Amount      decimal.Decimal `json:"amount" binding:"required"`
	Date        string          `json:"date" binding:"required"`
	Competence  string          `json:"competence" binding:"required"`
	Type        string          `json:"type" binding:"required"`
	Recurrent   bool            `json:"recurrent"`
}

type updateIncomeRequest struct {
	Description *string          `json:"description"`
	Amount      *decimal.Decimal `json:"amount"`
	Date        *string          `json:"date"`
	Type        *string          `json:"type"`
	Recurrent   *bool            `json:"recurrent"`
}

type propagateRequest struct {
	Competence string `json:"competence" binding:"required"`
}
