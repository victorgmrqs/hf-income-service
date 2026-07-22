> Part of the `testing-guide-hf-income-service` skill (see `../SKILL.md`).

# Response Envelope (`pkg/response/*.go`)

## What to test

- `Success(c, status, data)` produces `{"data": <data>, "error": null}` with the given status code.
- `Error(c, status, code, message)` produces `{"data": null, "error": {"code": ..., "message": ...}}` with the given status code.

This is a single-path utility with no branching, which the fundamentals normally say to skip at the unit layer — the exception here is that it's a **system boundary contract**: every single API response in the service depends on this exact shape, so a regression here breaks every client, not just one caller.

## Layer assignment

Unit — no external system involved; a plain `httptest.NewRecorder()` + `gin.Context` is enough, no need for a full router.

## Setup pattern

```go
package response_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/victorgmrqs/hf-income-service/src/pkg/response"
)

func TestSuccess_WritesEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	response.Success(c, 200, map[string]string{"id": "abc"})

	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["error"] != nil {
		t.Errorf("error = %v, want nil", body["error"])
	}
}
```

## When to skip

- Don't add a test per possible `data`/`status`/`code` value — the shape is what matters, not the specific payload.

## Examples from project

- `response_test.go` — covers `Success` and `Error` envelope shape.
