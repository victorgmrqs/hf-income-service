---
name: task
description: >
  Workflow completo de implementação a partir de um ticket Jira (hf-income-service).
  Lê o ticket via Atlassian MCP, carrega contexto, gera task-brief.yaml para
  aprovação (checkpoint 1), cria o branch de development e move o card para
  In Progress, implementa, testa, atualiza documentação (via /docs-sync), valida
  o DoD (checkpoint 2), executa /code-review-task, commita, faz push e move o
  card para In Review. PR aberto manualmente. Uso: /task <TICKET-ID>  ex: /task HF-42
---

# Skill: /task (hf-income-service)

## Objetivo

Executar o ciclo completo de desenvolvimento de um ticket Jira com dois checkpoints humanos e um gate de code review. Documentação e testes fazem parte da entrega — não são etapas posteriores. Processo canônico: [docs/workflow.md](../../../docs/workflow.md).

Projeto Jira: `HF` — https://goncalvesmarques.atlassian.net.
Colunas: `Backlog → In Progress (Fase 3.0) → In Review (Fase 8) → Done (manual, no merge)`.

> Tasks `[frontend]` pertencem ao repo hf-frontend; tasks de despesas/pagamentos ao hf-transaction-service. Se o ticket não for deste serviço, avise e pare.

---

## Fase 1 — Carregamento de contexto

### 1.1 Ticket Jira
Leia o ticket via Atlassian MCP. Extraia: título, descrição, tipo, critérios de aceite, comentários.

### 1.2 Contexto do projeto
Leia sempre: `CLAUDE.md`, `docs/workflow.md`, `docs/observability.md`.
Conforme o escopo do ticket:
```
docs/rules/REC.md + docs/fdds/fdd-001-rec.md   → receitas
docs/rules/ORC.md + docs/fdds/fdd-002-orc.md   → teto global
docs/rules/MET.md + docs/fdds/fdd-003-met.md   → metas de redução
docs/rules/SAL.md + docs/fdds/fdd-004-sal.md   → saldo/balance
INTEGRATIONS.md / openapi.yaml                  → chamada ao hf-transaction-service / contrato REST
```

### 1.3 Regras no Notion
Leia a página de regras do(s) domínio(s) via MCP. Workspace: https://www.notion.so/30ee65ad4ba98086809ed3a3f38ef45f (Notion prevalece sobre o espelho local).

---

## Fase 2 — task-brief.yaml

Gere `task-brief.yaml` na raiz (não versionado — já no `.gitignore`):

```yaml
ticket: {id: "", title: "", type: ""}      # feature | bug | chore | refactor
understanding: {problem: "", proposed_solution: ""}
scope:
  domains: []                # REC | ORC | SAL | MET
  affected_files: []         # arquivos Go
  business_rules: {modified: [], new: []}   # ex: REC-03 / REC-07
  cross_service_impact: false               # chama hf-transaction-service? (INTEGRATIONS.md)
  api_contract_change: false                # contrato REST muda? (openapi.yaml)
  db_migration_required: false              # schema GORM muda? (ENTITIES.md)
risks: {adr_required: false, ambiguities: []}
implementation_plan: {steps: []}
test_scenarios: []           # nomes exatos dos _test.go obrigatórios (ver Fase 4)
dod_checklist:
  - "[ ] critérios de aceite do ticket satisfeitos"
  - "[ ] testes unitários (mock de repo) criados/atualizados; cenários obrigatórios presentes e passando"
  - "[ ] testes de integração (testcontainers) criados/atualizados (se toca repositório)"
  - "[ ] go vet, gofmt e golangci-lint sem erros; go test ./... verde"
  - "[ ] docs/rules/<DOMAIN>.md + Notion atualizados (se regra adicionada/modificada)"
  - "[ ] openapi.yaml atualizado (se api_contract_change); INTEGRATIONS.md (se cross_service_impact)"
  - "[ ] IDs de regra referenciados nos erros Go (// DOMAIN-XX)"
  - "[ ] use case novo loga INFO na entrada (operation, user_id, competence) e saída (duration_ms)"
  - "[ ] violação de regra incrementa BusinessErrorsTotal{domain, rule_id}; trace_id em logs de erro/warn"
  - "[ ] nenhum dado sensível em log (senha, token, key, connection string, body)"
  - "[ ] evidência .http em docs/http/<domain>/ cobrindo a matriz de erros do FDD seção 6"
  - "[ ] CHANGELOG.md atualizado ([Unreleased], (HF-XX))"
  - "[ ] PR description inclui seção 'Regras Afetadas'"
```

### CHECKPOINT 1 — Aprovação do brief
Apresente problema, solução, escopo (domínios, regras, impactos), **cenários de teste obrigatórios**, ambiguidades e plano numerado. **Aguarde aprovação.** Nenhum brief é aprovado sem `test_scenarios`.

---

## Fase 3.0 — Branch + Jira In Progress

1. `git checkout development && git pull --ff-only` (pull só se houver remote) → `git checkout -b feat/HF-XX-descricao-curta`. Se `development` não existir, crie de `main` e avise. Nunca trabalhe em `main`/`development`.
2. Jira → **In Progress** via `getTransitionsForJiraIssue` + `transitionJiraIssue`. Se a coluna não existir, avise e siga.

---

## Fase 3 — Implementação

Regras obrigatórias (ver CLAUDE.md):
- Dependência estrita: `Handler → UseCase → Repository (interface) → DB`.
- Valores monetários: `decimal.Decimal` — nunca `float64`. UUIDs em `BeforeCreate` (`google/uuid`). Soft delete via `gorm.DeletedAt`.
- Responses sempre via `pkg/response` (`Success`/`Error`).
- Toda variável de erro que implementa regra: comentário `// DOMAIN-XX` acima.
- Use case novo: log `INFO` na entrada (operation, user_id, competence) e saída (duration_ms).
- Violação de regra: `metrics.BusinessErrorsTotal.WithLabelValues("DOMAIN","DOMAIN-XX").Inc()`; `slog.String("trace_id", traceID)` em erro/warn.
- Nunca logar senha/token/key/connection string/body.

- **`new_rules` não vazio:** documente a regra no Notion **antes** de referenciá-la no código; depois adicione em `docs/rules/<DOMAIN>.md`.
- **`api_contract_change: true`:** atualize `openapi.yaml` junto com o handler.
- **`db_migration_required: true`:** schema via GORM AutoMigrate; documente em `ENTITIES.md`.

---

## Fase 4 — Testes

- **Use cases:** unitários com mock das interfaces de repositório. **Repositórios:** integração com `testcontainers-go` (PostgreSQL real). **HTTP client:** mock via interface nos unit tests.
- Padrões de setup: [.claude/skills/testing-guide.md](../testing-guide.md) (ou `.claude/skills/testing-guide/`).
- Cenários obrigatórios por use case (nomes exatos):
  - `Test<UseCase>_Success`
  - `Test<UseCase>_MissingRequiredField` (quando aplicável)
  - `Test<UseCase>_<NomeDoErro>` — um por linha da matriz de erros do FDD seção 6
  - `Test<UseCase>_UpstreamError` — quando o use case chama o hf-transaction-service

```bash
gofmt -l . && go vet ./... && golangci-lint run && go test ./...
```

Após os testes passarem, crie a evidência `.http` em `docs/http/<domain>/<HF-XX>-<descricao>.http`, um request comentado por cenário (`### Cenário: ...`).

---

## Fase 5 — Documentação

Invoque **`/docs-sync <HF-XX>`** (via Skill tool): aplica a tabela "mudou X → atualiza doc Y" (CHANGELOG, rules, openapi, INTEGRATIONS, ENTITIES, ADR) e sincroniza regras de negócio com o Notion. Revise o relatório antes de seguir.

---

## Fase 6 — DoD (CHECKPOINT 2)
Verifique cada item do `dod_checklist` e apresente status PRONTO/PENDENTE. **Aguarde aprovação.**

## Fase 6.5 — Code Review (gate)
Invoque `/code-review-task <HF-XX>`. REPROVADO → corrija e repita. Só prossiga APROVADO.

## Fase 7 — PR Description
Preencha o body sobre `.github/pull_request_template.md` (não edite o template): Jira Issue, o que foi feito, como testar, **Regras Afetadas** (IDs criados/modificados), veredito do review.

## Fase 8 — Entrega
1. Commit `type(HF-XX): mensagem`; `task-brief.yaml` fora do commit.
2. `git push -u origin feat/HF-XX-...` (sem remote: avise e pare).
3. Jira → **In Review** + comentário com resumo, branch e PR description.
4. PR e merge manuais (base `development`); **Done** manual no merge.

---

## Restrições
- Nada de código antes do CHECKPOINT 1; nada de commit antes do review APROVADO.
- Nunca commit direto em `main`/`development`. Ambiguidade sem resposta: pare e pergunte.
