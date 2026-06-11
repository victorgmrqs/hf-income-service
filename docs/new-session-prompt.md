# Prompt de retomada — hf-income-service

> Cole este texto no início de uma nova conversa no Claude Code para retomar o contexto.

---

## Contexto do projeto

Estou desenvolvendo o `hf-income-service`, um Go REST API que faz parte do ecossistema Home Finance.
O serviço é responsável por: receitas (REC), orçamento global mensal (ORC), metas de redução de gastos (MET) e saldo mensal calculado (SAL).

**Diretório:** `/home/victor/Projects/personal/home-finance/hf-income-service`
**Jira:** https://goncalvesmarques.atlassian.net (projeto HF)
**Notion:** https://www.notion.so/30ee65ad4ba98086809ed3a3f38ef45f

Leia o `CLAUDE.md` do projeto antes de qualquer ação — ele é a fonte de verdade para arquitetura, observabilidade e DoD.

---

## O que foi feito na sessão anterior

### Documentação criada
- `docs/requirements.md` — RF e RNF por domínio com rastreabilidade
- `docs/fdds/fdd-001-rec.md` — FDD Receitas
- `docs/fdds/fdd-002-orc.md` — FDD Orçamento Global
- `docs/fdds/fdd-003-met.md` — FDD Metas de Redução
- `docs/fdds/fdd-004-sal.md` — FDD Saldo Mensal
- `docs/implementation-plan.md` — 19 tasks em 6 fases (~24 dias)
- `docs/workflow.md` — template de ticket, Epics, Labels, prefixos, convenções, guia para agentes IA
- `docs/rules/REC.md`, `ORC.md`, `SAL.md`, `MET.md` — regras de negócio por domínio

### Skills criadas
- `.claude/skills/fdd-creator.md` — entrevista estruturada para gerar FDDs
- `.claude/skills/implementation-plan-creator.md` — lê FDDs e gera plano de implementação
- `.claude/skills/task.md` — ciclo completo de desenvolvimento a partir de um ticket Jira

### Jira — estrutura criada

**Epics:**
- HF-50: Fase 0 — Infraestrutura base
- HF-51: Fase 1 — REC
- HF-52: Fase 2 — ORC
- HF-53: Fase 3 — MET
- HF-54: Fase 4 — SAL
- HF-55: Fase 5 — Wiring

**Tasks (19 no total):**

| Ticket | Task | Fase | Descrição |
|--------|------|------|-----------|
| HF-56 | T01 | 0 | config, database, response |
| HF-59 | T02 | 0 | Entidades GORM |
| HF-57 | T03 | 0 | pkg/httpclient |
| HF-58 | T04 | 0 | Servidor Gin base |
| HF-60 | T05 | 1 (REC) | IncomeRepository + testes integração |
| HF-61 | T06 | 1 (REC) | income use cases: create, get, list, delete |
| HF-62 | T07 | 1 (REC) | income use cases: update, propagate |
| HF-63 | T08 | 1 (REC) | income handler |
| HF-64 | T09 | 2 (ORC) | GlobalBudgetRepository + testes integração |
| HF-65 | T10 | 2 (ORC) | budget use cases: create, get, update |
| HF-66 | T11 | 2 (ORC) | budget use cases: auto_adjust, preview_next |
| HF-67 | T12 | 2 (ORC) | budget handler |
| HF-68 | T13 | 3 (MET) | ReductionGoalRepository + testes integração |
| HF-71 | T14 | 3 (MET) | goal CRUD use cases |
| HF-69 | T15 | 3 (MET) | goal comparison + close_month |
| HF-70 | T16 | 3 (MET) | goal handler |
| HF-74 | T17 | 4 (SAL) | balance use case (agregação paralela) |
| HF-72 | T18 | 4 (SAL) | balance handler |
| HF-73 | T19 | 5 (Wiring) | main.go + smoke tests |

> Nota: alguns tickets ficaram fora de ordem numérica porque a conexão MCP caiu durante a criação em lote e foi necessário recriar individualmente. A ordem lógica é a da tabela acima.

---

## Próximo passo sugerido

Iniciar a implementação com `/task HF-56` (T01 — Infraestrutura base: config, database, response).

O skill `/task HF-XX` executa o ciclo completo:
1. Lê o ticket via Atlassian MCP
2. Carrega o contexto (CLAUDE.md, FDD, regras)
3. Gera `task-brief.yaml` para aprovação (CHECKPOINT 1)
4. Implementa, testa, cria arquivo `.http` de evidência
5. Valida DoD (CHECKPOINT 2)
6. Gera PR description

---

## Convenções importantes a lembrar

- Todo título de ticket tem prefixo: `[backend]`, `[frontend]`, `[infra]`, `[observability]`, `[cross-service]`, `[database]`, `[docs]`
- Valores monetários: sempre `decimal.Decimal`, nunca `float64`
- Erros de regra: comentário `// DOMAIN-XX` acima da variável
- Nunca logar: passwords, tokens, bodies de request/response, connection strings
- Sempre incluir `trace_id` em log ERROR/WARN
- `BusinessErrorsTotal.WithLabelValues("DOMAIN", "DOMAIN-XX").Inc()` em toda violação de regra
- Testes de repositório: PostgreSQL real via `testcontainers-go` (não SQLite)
- Testes de use case: mocks das interfaces (não banco real)
