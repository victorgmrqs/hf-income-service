package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

func TestResponseSuccess_ReturnsCorrectShape(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, http.StatusOK, gin.H{"id": "abc"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// "error" deve ser explicitamente null e "data" deve estar presente.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("corpo não é JSON válido: %v", err)
	}
	if _, ok := raw["data"]; !ok {
		t.Error(`campo "data" ausente`)
	}
	if string(raw["error"]) != "null" {
		t.Errorf(`"error" = %s, esperado null`, raw["error"])
	}
}

func TestResponseError_ReturnsCorrectShape(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Error(c, http.StatusBadRequest, "REC-02", "valor inválido")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var body struct {
		Data  interface{} `json:"data"`
		Error *ErrorInfo  `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("corpo não é JSON válido: %v", err)
	}
	if body.Data != nil {
		t.Errorf(`"data" = %v, esperado nil`, body.Data)
	}
	if body.Error == nil {
		t.Fatal(`"error" ausente, esperado objeto {code, message}`)
	}
	if body.Error.Code != "REC-02" || body.Error.Message != "valor inválido" {
		t.Errorf("error = %+v, esperado {REC-02, valor inválido}", body.Error)
	}
}
