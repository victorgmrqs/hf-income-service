# Changelog

Todas as mudanças relevantes deste serviço são documentadas aqui.
Formato baseado em [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/).
Cada entrada referencia o ticket Jira (`(HF-XX)`). Datas em `YYYY-MM-DD`.

## [Unreleased]

### Added
- Gates de qualidade no CI: job `quality` (gofmt + go vet + golangci-lint só código novo do PR) e job `docs-guard` (em pull_request); `build-and-push` passa a depender de `quality`. (HF-91)
- `.golangci.yml` (errcheck, govet, ineffassign, staticcheck, unused) e `scripts/docs-guard.sh` (código em `internal/` sem CHANGELOG/teste no mesmo diff falha o PR). (HF-91)
- Skills do workflow versionadas: `/task`, `/code-review-task`, `/docs-sync` em `.claude/skills/<nome>/SKILL.md` (Claude Code) e `.agents/skills/<nome>.md` (agy); `task.md` plano antigo removido. (HF-96)

### Fixed
- `pkg/observability/tracing.go` reformatado com gofmt. (HF-91)

### Dependencies
- `go mod tidy`: requires indiretos sincronizados no `go.mod` (entradas faltantes de `gin` e transitivos). (HF-91)
