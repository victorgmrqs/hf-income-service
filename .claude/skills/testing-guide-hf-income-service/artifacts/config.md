> Part of the `testing-guide-hf-income-service` skill (see `../SKILL.md`).

# Config (`src/config/config.go`)

## What to test

- `Load()`'s one real branch: a missing `.env` file (`viper.ConfigFileNotFoundError`/`os.IsNotExist`) must NOT be treated as an error — only other read failures should propagate.
- Defaults: when no env var is set, `Load()` returns the documented defaults from `.env.example` (e.g. `APP_PORT=8081`, `SCHEDULER_ENABLED=true`).
- Env var override: setting an env var (`t.Setenv`) changes the corresponding `Config` field.

This is mostly framework passthrough (Viper does the actual parsing) — low priority per the fundamentals ("framework behavior" is NOT worth testing), but the missing-file-is-not-an-error branch is real logic worth one test if this file changes.

## Layer assignment

Unit — no real system involved; use `t.Setenv` (auto-restored after the test) rather than mutating the process environment directly.

## Setup pattern

```go
func TestLoad_MissingEnvFile_UsesDefaults(t *testing.T) {
	t.Setenv("APP_PORT", "") // ensure no leftover from the environment
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Port != "8081" {
		t.Errorf("Server.Port = %q, want default 8081", cfg.Server.Port)
	}
}
```

## When to skip

- Don't test that Viper reads env vars at all — that's Viper's own guarantee, not this package's logic.

## Examples from project

- No test exists yet — low priority; add the missing-file-not-an-error scenario if this file's error handling changes, otherwise it's acceptable to leave uncovered under this project's pragmatic coverage philosophy (`references/file-conventions.md`).
