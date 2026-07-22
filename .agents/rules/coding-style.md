---
paths:
  - "src/**/*.go"
---
# Coding Style

Only conventions `gofmt`/`go vet`/`golangci-lint` (errcheck, govet, ineffassign, staticcheck, unused) do not already enforce:

- Logging is always structured via `slog`: `logger.InfoContext(ctx, "<event>", slog.String("key", val), ...)` — never `fmt.Sprintf` into the message string, never `log.Printf`. The logger is a `*slog.Logger` field injected via constructor, never `slog.Default()` inside business code.
- Use case entry/exit log messages follow `"<domain>.<operation> started"` / `"<domain>.<operation> completed"`, with `slog.String("operation", "<domain>.<operation>")` and, on exit, `slog.Int64("duration_ms", time.Since(start).Milliseconds())`.
- Sentinel errors: one `var ErrXxx = errors.New("lowercase, no punctuation")` per line in `errors.go`, each preceded by a `// RULE-ID` comment when it enforces a business rule (omit the comment for purely technical errors like not-found).
- Monetary values are `decimal.Decimal` end-to-end (input DTO → domain → entity → repository) — never convert to `float64`, even for logging or intermediate math.
- Package-level doc comment (`// Package x contém ...`) only on packages whose purpose isn't obvious from the directory name (e.g. `handler/income`), in Portuguese, one line.
- Imports grouped in three blocks separated by a blank line: stdlib, third-party, internal (`github.com/victorgmrqs/hf-income-service/...`) — matches `goimports` default grouping already used throughout the repo.
- Small package-private helpers (date parsing, trace ID extraction, log helpers) live in a `helpers.go` per package rather than being duplicated inline or promoted to `pkg/`.
- Interface names are the capability, not `I`-prefixed (`CreateUseCase`, not `ICreateUseCase`); the private struct implementing it is the lowercase noun (`createUseCase`).
