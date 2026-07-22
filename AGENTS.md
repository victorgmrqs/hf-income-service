# AGENTS.md

> Arquivo de instruções cross-tool (Claude Code, Antigravity, Cursor).
> Instruções específicas por ferramenta: `CLAUDE.md` (Claude) | `GEMINI.md` (Antigravity).

## Overview

Home Finance Income Service — API REST em Go responsável por receitas, teto mensal de orçamento, cálculo de saldo mensal e metas de redução de gastos. Consome `hf-transaction-service` (despesas/contas a pagar) para compor saldo e comparativos.

## Stack

Go 1.25 · Gin (HTTP) · GORM + PostgreSQL · Viper (config) · testcontainers-go (integração) · Prometheus + OTEL (observabilidade).

## Commands

```bash
gofmt -l . && go vet ./... && golangci-lint run   # lint
go test ./...                                      # tests
go test -cover ./...                               # coverage
go build ./src/cmd/server                          # build
```

## Non-negotiable constraints

- Clean architecture: `Handler → UseCase → Repository (interface) → DB`; uma operação por arquivo em `usecase/<domain>/`.
- Toda validação/erro de regra de negócio referencia o rule ID em comentário Go (`// REC-03`).
- Todo use case novo loga INFO na entrada/saída; violação de regra loga WARN com `rule_id`; erro no handler loga ERROR com `trace_id`.
- `BusinessErrorsTotal{domain, rule_id}` incrementado em cada erro de regra de negócio.
- Nunca logar senha, token, API key, connection string, body de request/response ou headers `Authorization`/`OTEL_EXPORTER_OTLP_HEADERS`.
- Fonte de verdade das regras de negócio: **Confluence** (space `HF`); espelho local em `docs/rules/<DOMAIN>.md` — Confluence prevalece em divergência.
- Sem mock de `hf-transaction-service` em testes unitários — usar abstração de interface (`pkg/httpclient`).
- UUID v4 gerado em `BeforeCreate`; soft delete via `gorm.DeletedAt` (exceções documentadas em `docs/adr/`).

## Workflow

- Planejamento: `docs/requirements.md` → `docs/fdds/` → tickets Jira (`HF`).
- Execução: `/task <TICKET-ID>` → brief → aprovação → implementação/testes → docs → DoD → review → entrega.
- Configuração: `.dev-workflow/workflow.config.yaml`.

## Documentation loaded on demand

- Regras de negócio: `docs/rules/<DOMAIN>.md` (REC, ORC, SAL, MET)
- Arquitetura: `ARCHITECTURE.md`; ADRs: `docs/adr/`
- Domain design: `docs/fdds/`
- Contrato REST: `openapi.yaml` / `API_SPEC.md`; integrações: `INTEGRATIONS.md`
