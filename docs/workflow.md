# Workflow de Desenvolvimento — hf-income-service

> Espelho local. Fonte de verdade: Notion/Confluence.
> Notion: https://www.notion.so/30ee65ad4ba98086809ed3a3f38ef45f

---

## Visão Geral

Todo trabalho começa no Jira. Nenhum código é escrito sem um ticket aprovado. Agentes de IA usam o skill `/task <HF-XX>` para executar o ciclo completo de desenvolvimento a partir do ticket.

```
Jira ticket → /task HF-XX → task-brief.yaml (CHECKPOINT 1) → implementação + testes → docs → DoD (CHECKPOINT 2) → PR
```

---

## Estrutura do Jira

**Projeto:** `HF` — Home Finance
**URL:** https://goncalvesmarques.atlassian.net

### Tipos de Issue

| Tipo | Uso |
|------|-----|
| Epic | Agrupa tasks por fase de entrega ou funcionalidade transversal |
| Task | Unidade de implementação — um task do plano = um ticket |
| Feature | Nova capacidade do produto (gerada por produto, não por plano técnico) |
| Bug | Defeito em comportamento existente |
| Subtask | Parte menor de uma Task quando necessário dividir |

### Epics — hf-income-service

Cada fase do `docs/implementation-plan.md` tem um Epic correspondente. O prefixo `[hf-income-service]` permite filtrar por serviço em views multi-projeto.

| Epic | Fase | Escopo |
|------|------|--------|
| `[hf-income-service] Fase 0 — Infraestrutura base` | 0 | config, entities, httpclient, Gin base |
| `[hf-income-service] Fase 1 — REC` | 1 | Receitas (repository, use cases, handler) |
| `[hf-income-service] Fase 2 — ORC` | 2 | Orçamento Global (repository, use cases, handler) |
| `[hf-income-service] Fase 3 — MET` | 3 | Metas de Redução (repository, use cases, handler) |
| `[hf-income-service] Fase 4 — SAL` | 4 | Saldo Mensal (use case agregado, handler) |
| `[hf-income-service] Fase 5 — Wiring` | 5 | main.go, smoke tests, validação final |

> Epics podem ser compartilhados entre serviços quando a entrega impacta o frontend, hf-transaction-service ou um futuro auth service. Nesse caso, omitir o prefixo e usar um nome de produto (ex: `[Home Finance] Auth — JWT`).

### Labels

Labels são tags livres usadas para filtros e busca cross-projeto.

| Label | Quando usar |
|-------|-------------|
| `hf-income-service` | Sempre — identifica o serviço |
| `REC` / `ORC` / `SAL` / `MET` | Domínio de negócio afetado |
| `infra` | Tasks de infraestrutura sem domínio (Fase 0) |
| `cross-service` | Impacto em mais de um serviço |
| `observability` | Task específica de logs, métricas ou tracing |

---

## Template de Ticket

> Cole este template na **description** do ticket Jira ao criar manualmente.
> Ao criar via agente, o skill `/task` preenche automaticamente via `task-brief.yaml`.

```markdown
# Título: [prefixo] TXX — Descrição curta
> Prefixo obrigatório: [backend] | [frontend] | [infra] | [observability] | [cross-service] | [database] | [docs]
> Escolha o prefixo da camada PRINCIPAL afetada. Use labels para camadas secundárias.

## Contexto
> Por que esta task existe, qual FDD originou, qual fase e ID do plano.

FDD: `docs/fdds/fdd-XXX-yyy.md` | Fase: N | Plano: TXX

## Escopo — Arquivos
> Lista exata de arquivos a criar ou modificar. Sem ambiguidade.

- `internal/usecase/<domain>/create.go` — criar
- `internal/usecase/<domain>/create_test.go` — criar

## Regras de Negócio Afetadas
| ID | Regra | Arquivo |
|----|-------|---------|
| DOM-01 | [descrição resumida] | `docs/rules/DOM.md` |

## Critérios de Aceite
- [ ] [critério funcional verificável — binário]
- [ ] Testes unitários cobrem todos os cenários de erro da matriz (FDD seção 6)
- [ ] `BusinessErrorsTotal{domain="DOM", rule_id="DOM-XX"}` incrementado em cada erro
- [ ] Log INFO na entrada e saída do use case
- [ ] Arquivo `.http` de evidência criado em `docs/http/<domain>/`

## Cenários de Teste Obrigatórios
> Testes que DEVEM existir — não negociável. Nomear exatamente assim nos arquivos _test.go.

- `Test<UseCase>_Success` — caminho feliz
- `Test<UseCase>_MissingRequiredField` — campo obrigatório ausente
- `Test<UseCase>_InvalidAmount` — valor inválido (quando aplicável)
- `Test<UseCase>_NotFound` — entidade não encontrada (update/delete)
- `Test<UseCase>_UpstreamError` — falha no httpclient (quando aplicável)

## Evidência
> Arquivo `.http` com os cURLs dos cenários. Rodar antes de fechar o ticket.

`docs/http/<domain>/<task-id>-<descricao>.http`

## Dependências
- Bloqueado por: HF-XX (TXX — descrição)

## Referências
- FDD: `docs/fdds/fdd-XXX-yyy.md`
- Regras: `docs/rules/DOM.md`
- Observabilidade: `docs/observability.md`
- Padrões: `CLAUDE.md` (seções Architecture, Observability Conventions)
```

---

## Arquivos `.http` de Evidência

Cada ticket que toca um endpoint deve incluir um arquivo `.http` cobrindo todos os cenários da seção 6 do FDD (matriz de erros).

**Localização:** `docs/http/<domain>/<ticket-id>-<descricao>.http`

**Exemplo:** `docs/http/income/HF-42-create-income.http`

```http
### Cenário: criação bem-sucedida
POST http://localhost:8081/api/v1/income
Content-Type: application/json

{
  "user_id": "00000000-0000-0000-0000-000000000001",
  "description": "Salário junho",
  "amount": "5000.00",
  "date": "2026-06-05",
  "competence": "2026-06",
  "type": "SALARY",
  "recurrent": false
}

### Cenário: amount inválido (REC-02)
POST http://localhost:8081/api/v1/income
Content-Type: application/json

{
  "user_id": "00000000-0000-0000-0000-000000000001",
  "description": "Salário",
  "amount": "-100.00",
  "date": "2026-06-05",
  "competence": "2026-06",
  "type": "SALARY"
}

### Cenário: campo obrigatório ausente (REC-01)
POST http://localhost:8081/api/v1/income
Content-Type: application/json

{
  "description": "Sem user_id"
}
```

> **Compatibilidade:** arquivos `.http` funcionam nativamente no VS Code com a extensão [REST Client](https://marketplace.visualstudio.com/items?itemName=humao.rest-client) e no JetBrains IDEs.

---

## Ciclo de Vida do Ticket

```
Backlog → In Progress → In Review → Done
```

| Status | Quem move | Condição |
|--------|-----------|----------|
| Backlog → In Progress | Dev/Agente | Ao iniciar `/task HF-XX` |
| In Progress → In Review | Dev/Agente | CHECKPOINT 2 aprovado, PR aberta |
| In Review → Done | Dev | PR merged |

---

## Convenções de Nomenclatura

### Prefixo de tipo no título do ticket

Todo ticket Jira deve ter um prefixo entre colchetes antes do título, indicando a camada ou domínio técnico afetado. Isso permite filtrar e priorizar visualmente em qualquer board.

| Prefixo | Quando usar |
|---------|-------------|
| `[backend]` | Use case, repository, handler Go |
| `[frontend]` | Componentes UI, páginas, integrações React |
| `[infra]` | Docker, Kubernetes, CI/CD, configuração de ambiente |
| `[observability]` | Logs, métricas Prometheus, tracing OTEL, dashboards Grafana |
| `[cross-service]` | Integração entre serviços (ex: httpclient, contratos de API) |
| `[database]` | Migrations, índices, schema — quando não coberto por uma task de backend |
| `[docs]` | Apenas documentação sem código associado |

**Exemplos de título completo:**
- `[backend] T05 — IncomeRepository: implementação + testes de integração`
- `[cross-service] T11 — budget use cases: auto_adjust + preview_next`
- `[infra] T19 — Wiring: cmd/server/main.go + smoke tests`
- `[observability] Configurar dashboard SAL no Grafana Cloud`

> Quando um ticket impacta mais de uma camada (ex: backend + cross-service), use o prefixo da camada **principal**. Adicione o label correspondente para a camada secundária.

---

### Nomenclatura de artefatos

| Artefato | Formato | Exemplo |
|----------|---------|---------|
| Branch | `feat/HF-XX-descricao-curta` | `feat/HF-42-income-create` |
| Commit | `type(scope): mensagem` | `feat(income): add create use case` |
| PR title | `feat(HF-XX): descrição` | `feat(HF-42): income create use case with propagation` |

---

## Uso com Agentes de IA

O skill `/task HF-XX` executa o workflow completo:

1. Lê o ticket via Atlassian MCP
2. Carrega o FDD referenciado na description
3. Gera `task-brief.yaml` para aprovação (CHECKPOINT 1)
4. Implementa conforme os arquivos do Escopo
5. Escreve os testes dos Cenários Obrigatórios
6. Cria o arquivo `.http` de Evidência
7. Valida o DoD (CHECKPOINT 2)
8. Gera a PR description

> `task-brief.yaml` não deve ser versionado — adicionar ao `.gitignore`.
