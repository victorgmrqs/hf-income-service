package goal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	goalUseCase "github.com/victorgmrqs/hf-income-service/src/internal/usecase/goal"
	"github.com/victorgmrqs/hf-income-service/src/pkg/observability"
)

func init() { gin.SetMode(gin.TestMode) }

// Mocks manuais dos use cases — cada campo func customiza o cenário.
type mockCreateUC struct {
	fn func(ctx context.Context, in goalUseCase.CreateInput) (*goalUseCase.GoalOutput, error)
}

func (m *mockCreateUC) Execute(ctx context.Context, in goalUseCase.CreateInput) (*goalUseCase.GoalOutput, error) {
	return m.fn(ctx, in)
}

type mockListUC struct {
	fn func(ctx context.Context, in goalUseCase.ListInput) ([]goalUseCase.GoalOutput, error)
}

func (m *mockListUC) Execute(ctx context.Context, in goalUseCase.ListInput) ([]goalUseCase.GoalOutput, error) {
	return m.fn(ctx, in)
}

type mockUpdateUC struct {
	fn func(ctx context.Context, in goalUseCase.UpdateInput) (*goalUseCase.GoalOutput, error)
}

func (m *mockUpdateUC) Execute(ctx context.Context, in goalUseCase.UpdateInput) (*goalUseCase.GoalOutput, error) {
	return m.fn(ctx, in)
}

type mockDeleteUC struct {
	fn func(ctx context.Context, id uuid.UUID) error
}

func (m *mockDeleteUC) Execute(ctx context.Context, id uuid.UUID) error { return m.fn(ctx, id) }

type mockComparisonUC struct {
	fn func(ctx context.Context, in goalUseCase.ComparisonInput) ([]goalUseCase.ComparisonItemOutput, error)
}

func (m *mockComparisonUC) Execute(ctx context.Context, in goalUseCase.ComparisonInput) ([]goalUseCase.ComparisonItemOutput, error) {
	return m.fn(ctx, in)
}

type mockCloseMonthUC struct {
	fn func(ctx context.Context, in goalUseCase.CloseMonthInput) (*goalUseCase.CloseMonthOutput, error)
}

func (m *mockCloseMonthUC) Execute(ctx context.Context, in goalUseCase.CloseMonthInput) (*goalUseCase.CloseMonthOutput, error) {
	return m.fn(ctx, in)
}

type handlerMocks struct {
	create     *mockCreateUC
	list       *mockListUC
	update     *mockUpdateUC
	delete     *mockDeleteUC
	comparison *mockComparisonUC
	closeMonth *mockCloseMonthUC
}

func newTestRouter(m handlerMocks) *gin.Engine {
	metrics := observability.NewServiceMetrics("hf_income_goal_handler_test_" + uuid.NewString()[:8])
	h := NewGoalHandler(m.create, m.list, m.update, m.delete, m.comparison, m.closeMonth, metrics)
	r := gin.New()
	goals := r.Group("/api/v1/goals/reduction")
	{
		goals.POST("", h.Create)
		goals.GET("", h.List)
		goals.GET("/comparison", h.Comparison)
		goals.POST("/close-month", h.CloseMonth)
		goals.PUT("/:id", h.Update)
		goals.DELETE("/:id", h.Delete)
	}
	return r
}

func doRequest(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func errorCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("corpo não é JSON válido: %v", err)
	}
	if body.Error == nil {
		t.Fatal("esperava envelope de erro")
	}
	return body.Error.Code
}

func TestGoalHandler_Create_Success(t *testing.T) {
	goalID := uuid.New()
	m := handlerMocks{create: &mockCreateUC{fn: func(_ context.Context, in goalUseCase.CreateInput) (*goalUseCase.GoalOutput, error) {
		return &goalUseCase.GoalOutput{ID: goalID, UserID: in.UserID, CategoryID: in.CategoryID, Competence: in.Competence, TargetAmount: in.TargetAmount}, nil
	}}}
	r := newTestRouter(m)

	w := doRequest(r, http.MethodPost, "/api/v1/goals/reduction",
		`{"user_id":"`+uuid.NewString()+`","category_id":"`+uuid.NewString()+`","competence":"2026-07","target_amount":"400.00"}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", w.Code, w.Body.String())
	}
}

func TestGoalHandler_Create_InvalidBody_ValidationError(t *testing.T) {
	r := newTestRouter(handlerMocks{create: &mockCreateUC{fn: func(context.Context, goalUseCase.CreateInput) (*goalUseCase.GoalOutput, error) {
		t.Fatal("use case não deve ser chamado com body inválido")
		return nil, nil
	}}})

	w := doRequest(r, http.MethodPost, "/api/v1/goals/reduction", `{"user_id":""}`)

	if w.Code != http.StatusBadRequest || errorCode(t, w) != "VALIDATION_ERROR" {
		t.Fatalf("status/code = %d/%s, want 400/VALIDATION_ERROR", w.Code, errorCode(t, w))
	}
}

func TestGoalHandler_Create_InvalidUserID(t *testing.T) {
	r := newTestRouter(handlerMocks{create: &mockCreateUC{fn: func(context.Context, goalUseCase.CreateInput) (*goalUseCase.GoalOutput, error) {
		return nil, nil
	}}})

	w := doRequest(r, http.MethodPost, "/api/v1/goals/reduction",
		`{"user_id":"not-a-uuid","category_id":"`+uuid.NewString()+`","competence":"2026-07","target_amount":"400.00"}`)

	if w.Code != http.StatusBadRequest || errorCode(t, w) != "INVALID_USER_ID" {
		t.Fatalf("status/code = %d/%s, want 400/INVALID_USER_ID", w.Code, errorCode(t, w))
	}
}

func TestGoalHandler_Create_GoalAlreadyExists(t *testing.T) {
	r := newTestRouter(handlerMocks{create: &mockCreateUC{fn: func(context.Context, goalUseCase.CreateInput) (*goalUseCase.GoalOutput, error) {
		return nil, goalUseCase.ErrGoalAlreadyExists
	}}})

	w := doRequest(r, http.MethodPost, "/api/v1/goals/reduction",
		`{"user_id":"`+uuid.NewString()+`","category_id":"`+uuid.NewString()+`","competence":"2026-07","target_amount":"400.00"}`)

	if w.Code != http.StatusConflict || errorCode(t, w) != "GOAL_ALREADY_EXISTS" {
		t.Fatalf("status/code = %d/%s, want 409/GOAL_ALREADY_EXISTS", w.Code, errorCode(t, w))
	}
}

func TestGoalHandler_Create_InvalidTargetAmount(t *testing.T) {
	r := newTestRouter(handlerMocks{create: &mockCreateUC{fn: func(context.Context, goalUseCase.CreateInput) (*goalUseCase.GoalOutput, error) {
		return nil, goalUseCase.ErrInvalidTargetAmount
	}}})

	w := doRequest(r, http.MethodPost, "/api/v1/goals/reduction",
		`{"user_id":"`+uuid.NewString()+`","category_id":"`+uuid.NewString()+`","competence":"2026-07","target_amount":"-10.00"}`)

	if w.Code != http.StatusBadRequest || errorCode(t, w) != "INVALID_TARGET_AMOUNT" {
		t.Fatalf("status/code = %d/%s, want 400/INVALID_TARGET_AMOUNT", w.Code, errorCode(t, w))
	}
}

func TestGoalHandler_Create_InvalidCategoryID(t *testing.T) {
	r := newTestRouter(handlerMocks{create: &mockCreateUC{fn: func(context.Context, goalUseCase.CreateInput) (*goalUseCase.GoalOutput, error) {
		t.Fatal("use case não deve ser chamado com category_id inválido")
		return nil, nil
	}}})

	w := doRequest(r, http.MethodPost, "/api/v1/goals/reduction",
		`{"user_id":"`+uuid.NewString()+`","category_id":"not-a-uuid","competence":"2026-07","target_amount":"400.00"}`)

	if w.Code != http.StatusBadRequest || errorCode(t, w) != "INVALID_CATEGORY_ID" {
		t.Fatalf("status/code = %d/%s, want 400/INVALID_CATEGORY_ID", w.Code, errorCode(t, w))
	}
}

func TestGoalHandler_Create_UnexpectedError_InternalServerError(t *testing.T) {
	r := newTestRouter(handlerMocks{create: &mockCreateUC{fn: func(context.Context, goalUseCase.CreateInput) (*goalUseCase.GoalOutput, error) {
		return nil, context.DeadlineExceeded
	}}})

	w := doRequest(r, http.MethodPost, "/api/v1/goals/reduction",
		`{"user_id":"`+uuid.NewString()+`","category_id":"`+uuid.NewString()+`","competence":"2026-07","target_amount":"400.00"}`)

	if w.Code != http.StatusInternalServerError || errorCode(t, w) != "INTERNAL_SERVER_ERROR" {
		t.Fatalf("status/code = %d/%s, want 500/INTERNAL_SERVER_ERROR", w.Code, errorCode(t, w))
	}
}

func TestGoalHandler_Comparison_MissingCompetence(t *testing.T) {
	r := newTestRouter(handlerMocks{comparison: &mockComparisonUC{fn: func(context.Context, goalUseCase.ComparisonInput) ([]goalUseCase.ComparisonItemOutput, error) {
		return nil, goalUseCase.ErrMissingRequiredField
	}}})

	w := doRequest(r, http.MethodGet, "/api/v1/goals/reduction/comparison?user_id="+uuid.NewString(), "")

	if w.Code != http.StatusBadRequest || errorCode(t, w) != "MISSING_REQUIRED_FIELD" {
		t.Fatalf("status/code = %d/%s, want 400/MISSING_REQUIRED_FIELD", w.Code, errorCode(t, w))
	}
}

func TestGoalHandler_List_Success(t *testing.T) {
	m := handlerMocks{list: &mockListUC{fn: func(context.Context, goalUseCase.ListInput) ([]goalUseCase.GoalOutput, error) {
		return []goalUseCase.GoalOutput{{ID: uuid.New()}}, nil
	}}}
	r := newTestRouter(m)

	w := doRequest(r, http.MethodGet, "/api/v1/goals/reduction?user_id="+uuid.NewString()+"&competence=2026-07", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
}

func TestGoalHandler_List_InvalidUserID(t *testing.T) {
	r := newTestRouter(handlerMocks{})

	w := doRequest(r, http.MethodGet, "/api/v1/goals/reduction?user_id=abc&competence=2026-07", "")

	if w.Code != http.StatusBadRequest || errorCode(t, w) != "INVALID_USER_ID" {
		t.Fatalf("status/code = %d/%s, want 400/INVALID_USER_ID", w.Code, errorCode(t, w))
	}
}

func TestGoalHandler_Update_Success(t *testing.T) {
	m := handlerMocks{update: &mockUpdateUC{fn: func(_ context.Context, in goalUseCase.UpdateInput) (*goalUseCase.GoalOutput, error) {
		return &goalUseCase.GoalOutput{ID: in.ID, TargetAmount: in.TargetAmount}, nil
	}}}
	r := newTestRouter(m)

	w := doRequest(r, http.MethodPut, "/api/v1/goals/reduction/"+uuid.NewString(), `{"target_amount":"350.00"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
}

func TestGoalHandler_Update_InvalidID(t *testing.T) {
	r := newTestRouter(handlerMocks{})

	w := doRequest(r, http.MethodPut, "/api/v1/goals/reduction/not-a-uuid", `{"target_amount":"350.00"}`)

	if w.Code != http.StatusBadRequest || errorCode(t, w) != "INVALID_ID" {
		t.Fatalf("status/code = %d/%s, want 400/INVALID_ID", w.Code, errorCode(t, w))
	}
}

func TestGoalHandler_Update_NotFound(t *testing.T) {
	r := newTestRouter(handlerMocks{update: &mockUpdateUC{fn: func(context.Context, goalUseCase.UpdateInput) (*goalUseCase.GoalOutput, error) {
		return nil, goalUseCase.ErrGoalNotFound
	}}})

	w := doRequest(r, http.MethodPut, "/api/v1/goals/reduction/"+uuid.NewString(), `{"target_amount":"350.00"}`)

	if w.Code != http.StatusNotFound || errorCode(t, w) != "GOAL_NOT_FOUND" {
		t.Fatalf("status/code = %d/%s, want 404/GOAL_NOT_FOUND", w.Code, errorCode(t, w))
	}
}

func TestGoalHandler_Delete_Success(t *testing.T) {
	r := newTestRouter(handlerMocks{delete: &mockDeleteUC{fn: func(context.Context, uuid.UUID) error { return nil }}})

	w := doRequest(r, http.MethodDelete, "/api/v1/goals/reduction/"+uuid.NewString(), "")

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Errorf("204 não deve ter corpo, veio: %s", w.Body.String())
	}
}

func TestGoalHandler_Delete_NotFound(t *testing.T) {
	r := newTestRouter(handlerMocks{delete: &mockDeleteUC{fn: func(context.Context, uuid.UUID) error {
		return goalUseCase.ErrGoalNotFound
	}}})

	w := doRequest(r, http.MethodDelete, "/api/v1/goals/reduction/"+uuid.NewString(), "")

	if w.Code != http.StatusNotFound || errorCode(t, w) != "GOAL_NOT_FOUND" {
		t.Fatalf("status/code = %d/%s, want 404/GOAL_NOT_FOUND", w.Code, errorCode(t, w))
	}
}

func TestGoalHandler_Comparison_Success(t *testing.T) {
	pct := -22.5
	m := handlerMocks{comparison: &mockComparisonUC{fn: func(context.Context, goalUseCase.ComparisonInput) ([]goalUseCase.ComparisonItemOutput, error) {
		return []goalUseCase.ComparisonItemOutput{{
			CategoryID:   uuid.New(),
			TargetAmount: decimal.RequireFromString("700.00"),
			VariationPct: &pct,
		}}, nil
	}}}
	r := newTestRouter(m)

	w := doRequest(r, http.MethodGet, "/api/v1/goals/reduction/comparison?user_id="+uuid.NewString()+"&competence=2026-07", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"variation_pct":-22.5`) {
		t.Errorf("variation_pct numérico esperado no JSON, veio: %s", w.Body.String())
	}
}

func TestGoalHandler_Comparison_InvalidUserID(t *testing.T) {
	r := newTestRouter(handlerMocks{})

	w := doRequest(r, http.MethodGet, "/api/v1/goals/reduction/comparison?user_id=zz&competence=2026-07", "")

	if w.Code != http.StatusBadRequest || errorCode(t, w) != "INVALID_USER_ID" {
		t.Fatalf("status/code = %d/%s, want 400/INVALID_USER_ID", w.Code, errorCode(t, w))
	}
}

func TestGoalHandler_Comparison_InvalidCompetence(t *testing.T) {
	r := newTestRouter(handlerMocks{comparison: &mockComparisonUC{fn: func(context.Context, goalUseCase.ComparisonInput) ([]goalUseCase.ComparisonItemOutput, error) {
		return nil, goalUseCase.ErrInvalidCompetence
	}}})

	w := doRequest(r, http.MethodGet, "/api/v1/goals/reduction/comparison?user_id="+uuid.NewString()+"&competence=2026-13", "")

	if w.Code != http.StatusBadRequest || errorCode(t, w) != "INVALID_COMPETENCE" {
		t.Fatalf("status/code = %d/%s, want 400/INVALID_COMPETENCE", w.Code, errorCode(t, w))
	}
}

func TestGoalHandler_CloseMonth_Success(t *testing.T) {
	m := handlerMocks{closeMonth: &mockCloseMonthUC{fn: func(context.Context, goalUseCase.CloseMonthInput) (*goalUseCase.CloseMonthOutput, error) {
		return &goalUseCase.CloseMonthOutput{Closed: 4}, nil
	}}}
	r := newTestRouter(m)

	w := doRequest(r, http.MethodPost, "/api/v1/goals/reduction/close-month", `{"competence":"2026-06"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"closed":4`) {
		t.Errorf("closed=4 esperado no JSON, veio: %s", w.Body.String())
	}
}

func TestGoalHandler_CloseMonth_MissingCompetence(t *testing.T) {
	r := newTestRouter(handlerMocks{closeMonth: &mockCloseMonthUC{fn: func(context.Context, goalUseCase.CloseMonthInput) (*goalUseCase.CloseMonthOutput, error) {
		t.Fatal("use case não deve ser chamado com body inválido")
		return nil, nil
	}}})

	w := doRequest(r, http.MethodPost, "/api/v1/goals/reduction/close-month", `{}`)

	if w.Code != http.StatusBadRequest || errorCode(t, w) != "VALIDATION_ERROR" {
		t.Fatalf("status/code = %d/%s, want 400/VALIDATION_ERROR", w.Code, errorCode(t, w))
	}
}
