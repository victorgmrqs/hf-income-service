---
paths:
  - "src/internal/usecase/**/*.go"
  - "src/internal/handler/**/*.go"
---
# Error Handling

- Every domain error is a package-level sentinel in `usecase/<domain>/errors.go`: `var ErrXxx = errors.New("...")`. Business-rule errors carry the rule ID as a comment directly above (`// REC-02`); technical/not-found errors (e.g. `ErrIncomeNotFound`) don't need one.
- Usecases return the sentinel directly — never wrap with `fmt.Errorf("...: %w", err)` before it reaches the handler; the handler matches with `errors.Is`.
- Before returning a rule-violation error, call the domain's `logRuleViolation(ctx, logger, ruleID, reason)` helper (WARN + `trace_id` + `rule_id`).
- The handler owns a single `handleError(c *gin.Context, err error)` method with one `switch { case errors.Is(err, ...): ... }` branch per sentinel. Each branch: increments `metrics.BusinessErrorsTotal.WithLabelValues(domain, ruleID).Inc()` (only for rule violations, not for not-found/technical errors), then calls `response.Error(c, httpStatus, "SCREAMING_SNAKE_CODE", err.Error())`.
- Unmatched errors fall through to a `default` branch: `http.StatusInternalServerError`, code `"INTERNAL_SERVER_ERROR"`.
- Never swallow an error silently (`_ = err`) outside test helpers — `.golangci.yml` enforces `errcheck` on non-test files already; don't add manual exceptions.

Example (real, `handler/income/income.go`):

```go
case errors.Is(err, incomeUseCase.ErrInvalidAmount):
    h.metrics.BusinessErrorsTotal.WithLabelValues("REC", "REC-02").Inc()
    response.Error(c, http.StatusBadRequest, "INVALID_AMOUNT", err.Error())
```
