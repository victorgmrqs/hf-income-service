package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	balancehandler "github.com/victorgmrqs/hf-income-service/src/internal/handler/balance"
	globalbudgethandler "github.com/victorgmrqs/hf-income-service/src/internal/handler/global_budget"
	goalhandler "github.com/victorgmrqs/hf-income-service/src/internal/handler/goal"
	incomehandler "github.com/victorgmrqs/hf-income-service/src/internal/handler/income"
	"github.com/victorgmrqs/hf-income-service/src/pkg/observability"
)

func init() { gin.SetMode(gin.TestMode) }

func TestHealth_ReturnsOK(t *testing.T) {
	logger := observability.NewLogger("test")
	metrics := observability.NewServiceMetrics("hf_income_router_test")
	// Use cases nil: o teste de /health não aciona rotas de domínio.
	incomeHandler := incomehandler.NewIncomeHandler(nil, nil, nil, nil, nil, nil, metrics)
	budgetHandler := globalbudgethandler.NewGlobalBudgetHandler(nil, nil, nil, nil, nil, metrics)
	balanceHandler := balancehandler.NewBalanceHandler(nil, metrics)
	goalHandler := goalhandler.NewGoalHandler(nil, nil, nil, nil, nil, nil, metrics)
	router := SetupRouter(logger, metrics, incomeHandler, budgetHandler, balanceHandler, goalHandler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("corpo não é JSON válido: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf(`status = %q, esperado "ok"`, body["status"])
	}
}

func TestRouter_GoalsReductionRoutesRegistered(t *testing.T) {
	logger := observability.NewLogger("test")
	metrics := observability.NewServiceMetrics("hf_income_router_goal_test")
	incomeHandler := incomehandler.NewIncomeHandler(nil, nil, nil, nil, nil, nil, metrics)
	budgetHandler := globalbudgethandler.NewGlobalBudgetHandler(nil, nil, nil, nil, nil, metrics)
	balanceHandler := balancehandler.NewBalanceHandler(nil, metrics)
	goalHandler := goalhandler.NewGoalHandler(nil, nil, nil, nil, nil, nil, metrics)
	router := SetupRouter(logger, metrics, incomeHandler, budgetHandler, balanceHandler, goalHandler)

	want := map[string]string{
		"POST /api/v1/goals/reduction":             "",
		"GET /api/v1/goals/reduction":              "",
		"GET /api/v1/goals/reduction/comparison":   "",
		"POST /api/v1/goals/reduction/close-month": "",
		"PUT /api/v1/goals/reduction/:id":          "",
		"DELETE /api/v1/goals/reduction/:id":       "",
	}
	for _, r := range router.Routes() {
		delete(want, r.Method+" "+r.Path)
	}
	if len(want) > 0 {
		t.Errorf("rotas MET não registradas: %v", want)
	}
}
