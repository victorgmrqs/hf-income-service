package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	balancehandler "github.com/victorgmrqs/hf-income-service/src/internal/handler/balance"
	globalbudgethandler "github.com/victorgmrqs/hf-income-service/src/internal/handler/global_budget"
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
	router := SetupRouter(logger, metrics, incomeHandler, budgetHandler, balanceHandler)

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
