package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	globalbudgethandler "github.com/victorgmrqs/hf-income-service/src/internal/handler/global_budget"
	incomehandler "github.com/victorgmrqs/hf-income-service/src/internal/handler/income"
	"github.com/victorgmrqs/hf-income-service/src/pkg/observability"
)

func init() { gin.SetMode(gin.TestMode) }

var (
	testMetrics *observability.ServiceMetrics
	testLogger  = observability.NewLogger("test")
)

func getTestMetrics() *observability.ServiceMetrics {
	if testMetrics == nil {
		testMetrics = observability.NewServiceMetrics("hf_income_router_test")
	}
	return testMetrics
}

func TestHealth_ReturnsOK(t *testing.T) {
	metrics := getTestMetrics()
	// Use cases nil: o teste de /health não aciona rotas de domínio.
	incomeHandler := incomehandler.NewIncomeHandler(nil, nil, nil, nil, nil, nil, metrics)
	budgetHandler := globalbudgethandler.NewGlobalBudgetHandler(nil, nil, nil, metrics)
	router := SetupRouter(testLogger, metrics, incomeHandler, budgetHandler)

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

func TestCORS_OPTIONS_Returns204AndHeaders(t *testing.T) {
	metrics := getTestMetrics()
	incomeHandler := incomehandler.NewIncomeHandler(nil, nil, nil, nil, nil, nil, metrics)
	budgetHandler := globalbudgethandler.NewGlobalBudgetHandler(nil, nil, nil, metrics)
	router := SetupRouter(testLogger, metrics, incomeHandler, budgetHandler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/health", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}

	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:5173" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", origin, "http://localhost:5173")
	}
	if credentials := w.Header().Get("Access-Control-Allow-Credentials"); credentials != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want %q", credentials, "true")
	}
	if methods := w.Header().Get("Access-Control-Allow-Methods"); methods == "" {
		t.Errorf("Access-Control-Allow-Methods is empty")
	}
}


