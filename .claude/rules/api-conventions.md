---
paths:
  - "src/internal/handler/**/*.go"
  - "src/pkg/response/**/*.go"
---
# API Conventions

- Every response (except `GET /health`) goes through the standardized envelope: `response.Success(c, status, data)` → `{"data": ..., "error": null}`; `response.Error(c, status, code, message)` → `{"data": null, "error": {"code", "message"}}`. Never `c.JSON(...)` directly from a domain handler.
- Error `code` is `SCREAMING_SNAKE_CASE` and distinct from the business `rule_id` (e.g. code `INVALID_AMOUNT` for rule `REC-02`) — the code identifies the HTTP-facing error type, the rule_id identifies the business rule.
- Status codes by verb: `POST` create → `201`; `GET` → `200`; `PUT` → `200`; `DELETE` → `204` with `nil` data; validation/parse failures → `400`; not-found → `404`; upstream (hf-transaction-service) failures → `502`/`503` per `INTEGRATIONS.md`.
- Routes are grouped under `/api/v1/<resource>` in `handler/router.go` via `router.Group(...)`, one `{ }` block per domain. Literal sub-paths (e.g. `/preview-next`) must be registered **before** `/:id` in the same group to avoid Gin routing them as an ID param.
- Path/query params are parsed and validated at the top of the handler method (`uuid.Parse`, required query params) before calling the usecase — never pass raw strings into `Execute`.
- Request bodies are dedicated `xxxRequest` structs (not the entity) with `json` + `binding:"required"` tags, bound via `c.ShouldBindJSON(&body)`; a bind error maps to `400 VALIDATION_ERROR` with `err.Error()` as message.
- Optional/PATCH-like fields on update requests are pointers (`*string`, `*decimal.Decimal`) so "not sent" is distinguishable from a zero value.

Example (real, `router.go`):

```go
budgets := api.Group("/budgets/global")
{
    budgets.POST("", globalBudgetHandler.Create)
    budgets.GET("/preview-next", globalBudgetHandler.PreviewNext) // antes de /:id
    budgets.PUT("/:id", globalBudgetHandler.Update)
}
```
