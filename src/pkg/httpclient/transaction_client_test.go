package httpclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/victorgmrqs/hf-income-service/src/pkg/httpclient"
)

func jsonHandler(status int, body any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}
}

func TestGetExpenseTotals_Success(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(http.StatusOK, map[string]any{
		"data": map[string]any{
			"competence":     "2026-06",
			"total_personal": 1800.00,
			"total_shared":   1400.00,
			"total_general":  3200.00,
		},
		"error": nil,
	}))
	defer srv.Close()

	client := httpclient.NewTransactionClient(srv.URL, srv.Client())
	out, err := client.GetExpenseTotals(context.Background(), "user-1", "2026-06")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatal("expected non-nil output")
	}
	if out.Competence != "2026-06" {
		t.Errorf("competence = %q, want %q", out.Competence, "2026-06")
	}
	wantGeneral := "3200"
	if out.TotalGeneral.String() != wantGeneral {
		t.Errorf("total_general = %s, want %s", out.TotalGeneral, wantGeneral)
	}
}

func TestGetExpenseTotals_Timeout_ReturnsUpstreamTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()

	// Inject a client with a very short timeout to trigger ErrUpstreamTimeout
	// before the internal 5s per-call limit fires.
	shortClient := &http.Client{Timeout: 10 * time.Millisecond}
	client := httpclient.NewTransactionClient(srv.URL, shortClient)

	_, err := client.GetExpenseTotals(context.Background(), "user-1", "2026-06")
	if !errors.Is(err, httpclient.ErrUpstreamTimeout) {
		t.Fatalf("err = %v, want ErrUpstreamTimeout", err)
	}
}

func TestGetExpenseTotals_UnexpectedStatus_ReturnsUpstreamError(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(http.StatusInternalServerError, map[string]any{
		"data":  nil,
		"error": map[string]string{"code": "INTERNAL", "message": "boom"},
	}))
	defer srv.Close()

	client := httpclient.NewTransactionClient(srv.URL, srv.Client())
	_, err := client.GetExpenseTotals(context.Background(), "user-1", "2026-06")
	if !errors.Is(err, httpclient.ErrUpstreamError) {
		t.Fatalf("err = %v, want ErrUpstreamError", err)
	}
}

func TestGetAccountsPayable_Success(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(http.StatusOK, map[string]any{
		"data": []map[string]any{
			{
				"id":          "bill-1",
				"description": "Escola",
				"amount":      850.00,
				"due_date":    "2026-06-10T00:00:00Z",
				"status":      "PENDING",
				"recurrence":  "MONTHLY",
				"paid_at":     nil,
				"expense_id":  nil,
			},
		},
		"error": nil,
	}))
	defer srv.Close()

	client := httpclient.NewTransactionClient(srv.URL, srv.Client())
	bills, err := client.GetAccountsPayable(context.Background(), "user-1", "2026-06-30")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bills) != 1 {
		t.Fatalf("len(bills) = %d, want 1", len(bills))
	}
	if bills[0].Description != "Escola" {
		t.Errorf("description = %q, want %q", bills[0].Description, "Escola")
	}
	wantAmount := "850"
	if bills[0].Amount.String() != wantAmount {
		t.Errorf("amount = %s, want %s", bills[0].Amount, wantAmount)
	}
}

func TestGetExpensesByCategory_Success(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path + "?" + r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		// Shape real de GET /expenses/totals/by-category (CAL-05): lista com
		// category_name e valores decimais serializados como string.
		_, _ = w.Write([]byte(`{"data":[
			{"category_id":"cat-1","category_name":"Alimentação","total":"500.00","percentage":"62.50"},
			{"category_id":"cat-2","category_name":"Transporte","total":"300.00","percentage":"37.50"}
		],"error":null}`))
	}))
	defer srv.Close()

	client := httpclient.NewTransactionClient(srv.URL, srv.Client())
	out, err := client.GetExpensesByCategory(context.Background(), "user-1", "cat-1", "2026-06")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantPath := "/api/v1/expenses/totals/by-category?user_id=user-1&competence=2026-06"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if out == nil {
		t.Fatal("expected non-nil output")
	}
	if out.CategoryID != "cat-1" {
		t.Errorf("category_id = %q, want %q", out.CategoryID, "cat-1")
	}
	if out.CategoryName != "Alimentação" {
		t.Errorf("category_name = %q, want %q", out.CategoryName, "Alimentação")
	}
	if out.Total.String() != "500" {
		t.Errorf("total = %s, want 500", out.Total)
	}
}

func TestGetExpensesByCategory_CategoryWithoutExpenses_ReturnsZero(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(http.StatusOK, map[string]any{
		"data":  []any{},
		"error": nil,
	}))
	defer srv.Close()

	client := httpclient.NewTransactionClient(srv.URL, srv.Client())
	out, err := client.GetExpensesByCategory(context.Background(), "user-1", "cat-9", "2026-06")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil || !out.Total.IsZero() {
		t.Fatalf("out = %+v, want total zero para categoria sem despesas", out)
	}
	if out.CategoryID != "cat-9" {
		t.Errorf("category_id = %q, want %q", out.CategoryID, "cat-9")
	}
}

func TestLastDayOfMonth_June(t *testing.T) {
	got, err := httpclient.LastDayOfMonth("2026-06")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "2026-06-30" {
		t.Errorf("got %q, want %q", got, "2026-06-30")
	}
}

func TestLastDayOfMonth_February_NonLeapYear(t *testing.T) {
	got, err := httpclient.LastDayOfMonth("2026-02")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "2026-02-28" {
		t.Errorf("got %q, want %q", got, "2026-02-28")
	}
}

func TestLastDayOfMonth_February_LeapYear(t *testing.T) {
	got, err := httpclient.LastDayOfMonth("2024-02")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "2024-02-29" {
		t.Errorf("got %q, want %q", got, "2024-02-29")
	}
}

func TestLastDayOfMonth_December(t *testing.T) {
	got, err := httpclient.LastDayOfMonth("2026-12")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "2026-12-31" {
		t.Errorf("got %q, want %q", got, "2026-12-31")
	}
}
