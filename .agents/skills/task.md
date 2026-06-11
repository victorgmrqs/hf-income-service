---
name: task
description: >
  Workflow completo de implementação a partir de um ticket Jira.
  Lê o ticket via Atlassian MCP, carrega contexto do projeto,
  gera um task-brief.yaml para aprovação (checkpoint 1), implementa,
  testa, atualiza documentação e valida o DoD (checkpoint 2).
  Uso: /task <TICKET-ID>   ex: /task HF-42
---

# Skill: /task

## Objetivo

Executar o ciclo completo de desenvolvimento para um ticket Jira com dois checkpoints de aprovação humana. Documentação e testes fazem parte da entrega — não são etapas posteriores.

---

## Fase 1 — Carregamento de contexto

### 1.1 Ticket Jira
Leia o ticket via Atlassian MCP usando o ID fornecido como argumento.
Extraia: título, descrição, tipo, critérios de aceite, comentários relevantes.

### 1.2 Contexto do projeto
Leia sempre:
```
AGENTS.md
ARCHITECTURE.md
```

Leia conforme o escopo do ticket:
```
docs/rules/REC.md     → se envolve receitas
docs/rules/ORC.md     → se envolve teto global
docs/rules/SAL.md     → se envolve saldo/balance
docs/rules/MET.md     → se envolve metas de redução
INTEGRATIONS.md       → se envolve chamada ao hf-transaction-service
openapi.yaml          → se pode haver mudança de contrato de API
```

### 1.3 Regras de negócio no Confluence
Acesse o Confluence via MCP. Leia a página de regras do(s) domínio(s) identificado(s) no ticket.
Confluence: https://goncalvesmarques.atlassian.net/wiki (space: Home Finance)

---

## Fase 2 — task-brief.yaml

Gere o arquivo `task-brief.yaml` na raiz do projeto:

```yaml
ticket:
  id: ""
  title: ""
  type: ""                  # feature | bug | chore | refactor

understanding:
  problem: ""               # O que precisa ser resolvido (1-3 frases)
  proposed_solution: ""     # Como será resolvido (1-3 frases)

scope:
  domains: []               # REC | ORC | SAL | MET
  affected_files: []        # Arquivos Go que serão modificados/criados
  business_rules:
    modified: []            # IDs de regras existentes que mudam (ex: REC-03)
    new: []                 # Novos IDs a criar (ex: REC-07)
  cross_service_impact: false
  api_contract_change: false
  db_migration_required: false

risks:
  adr_required: false
  ambiguities: []           # Dúvidas que precisam de resposta antes de implementar

implementation_plan:
  steps: []                 # Passos ordenados por dependência técnica

dod_checklist:
  - "[ ] testes unitários criados ou atualizados"
  - "[ ] testes de integração criados ou atualizados (se toca repositório)"
  - "[ ] docs/rules/<DOMAIN>.md atualizado (se regra adicionada/modificada)"
  - "[ ] Confluence atualizado (se regra adicionada/modificada)"
  - "[ ] openapi.yaml atualizado (se api_contract_change: true)"
  - "[ ] INTEGRATIONS.md atualizado (se cross_service_impact: true)"
  - "[ ] IDs de regra referenciados nos erros Go (// DOMAIN-XX)"
  - "[ ] novo use case tem log INFO na entrada e saída"
  - "[ ] violações de regra incrementam BusinessErrorsTotal com domain e rule_id"
  - "[ ] nenhum dado sensível referenciado em log statements"
  - "[ ] arquivo .http de evidência criado em docs/http/<domain>/ cobrindo todos os cenários da matriz de erros (FDD seção 6)"
  - "[ ] PR description inclui seção 'Regras Afetadas'"
```

### CHECKPOINT 1 — Aprovação do brief

Apresente ao usuário:

```
## Task Brief — <ID>

**Problema:** <understanding.problem>
**Solução:** <understanding.proposed_solution>

**Escopo:**
- Domínios: <domains>
- Regras modificadas: <business_rules.modified || "nenhuma">
- Novas regras: <business_rules.new || "nenhuma">
- Impacto cross-service: <cross_service_impact>
- Mudança de API: <api_contract_change>
- Migration necessária: <db_migration_required>

**Ambiguidades:** <ambiguities || "nenhuma">

**Plano:**
<implementation_plan.steps numerado>

Aprova? ("ok" para prosseguir ou corrija diretamente)
```

**Aguarde resposta antes de continuar.**
Se houver correções, atualize o `task-brief.yaml` e confirme antes de implementar.

---

## Fase 3 — Implementação

Siga o plano aprovado. Regras obrigatórias durante a implementação:

- Dependência estrita: `Handler → UseCase → Repository interface → DB`
- Valores monetários: `decimal.Decimal` — nunca `float64`
- UUIDs: gerados em `BeforeCreate` via `google/uuid`
- Soft delete: `gorm.DeletedAt` nas entidades
- Responses: sempre `response.Success()` ou `response.Error()` de `pkg/response`
- Toda variável de erro que implementa uma regra: comentário `// DOMAIN-XX` acima
- Todo use case novo: log `INFO` na entrada (operation, user_id, competence) e na saída (duration_ms)
- Toda violação de regra de negócio: `metrics.BusinessErrorsTotal.WithLabelValues("DOMAIN", "DOMAIN-XX").Inc()`
- Nunca logar: passwords, tokens, API keys, connection strings, bodies de request/response
- Sempre incluir `trace_id` nos log statements de erro e warning

**Se `new_rules` não está vazio:**
Documente a nova regra no Confluence via MCP antes de implementar o código que a referencia.
Depois adicione a linha em `docs/rules/<DOMAIN>.md`.

**Se `api_contract_change: true`:**
Atualize `openapi.yaml` junto com o handler — não deixe para depois.

**Se `db_migration_required: true`:**
O schema é gerenciado por GORM AutoMigrate. Documente as mudanças de schema em `ENTITIES.md`.

---

## Fase 4 — Testes

Siga as convenções do AGENTS.md:

- **Use cases:** testes unitários com mocks das interfaces de repositório
- **Repositórios:** testes de integração com `testcontainers-go` (PostgreSQL real)
- **HTTP client:** coberto por interface — mock nos unit tests do use case que o usa

Consulte `.agents/skills/testing-guide.md` (ou `.claude/skills/testing-guide.md`) para padrões de setup.

Cada use case novo ou modificado deve ter arquivo `_test.go` correspondente com os seguintes cenários obrigatórios (conforme `docs/workflow.md`):

- `Test<UseCase>_Success` — caminho feliz
- `Test<UseCase>_MissingRequiredField` — campo obrigatório ausente (quando aplicável)
- `Test<UseCase>_<NomeDoErro>` — um teste por linha da matriz de erros do FDD seção 6
- `Test<UseCase>_UpstreamError` — falha no httpclient (quando o use case chama hf-transaction-service)

**Após os testes passarem**, crie o arquivo `.http` de evidência:

```
docs/http/<domain>/<TICKET-ID>-<descricao>.http
```

Cubra cada cenário da matriz de erros do FDD com um request separado e comentado (`### Cenário: ...`).
Compatível com REST Client (VS Code) e JetBrains HTTP Client.

---

## Fase 5 — Atualização de documentação

Execute apenas o que se aplica ao ticket:

| Condição | Ação |
|----------|------|
| Nova regra | Adicionar linha em `docs/rules/<DOMAIN>.md` + Confluence via MCP |
| Regra modificada | Atualizar linha em `docs/rules/<DOMAIN>.md` + Confluence via MCP |
| Novo endpoint ou campo de response | Atualizar `openapi.yaml` |
| Nova coluna ou tabela | Atualizar `ENTITIES.md` |
| Mudança no contrato com hf-transaction-service | Atualizar `INTEGRATIONS.md` |
| Decisão arquitetural relevante | Criar `docs/adr/ADR-XXX-titulo.md` |

---

## Fase 6 — Validação do DoD

### CHECKPOINT 2 — Aprovação final

Verifique cada item do `dod_checklist` e apresente:

```
## DoD — <ID>

- [x] testes unitários criados
- [x] docs/rules/REC.md atualizado
- [ ] Confluence atualizado  ← PENDENTE
...

Status: PRONTO / PENDENTE (liste o que falta)
```

**Aguarde aprovação antes de gerar a PR description.**

---

## Fase 7 — PR Description

Gere o body da PR seguindo `.github/pull_request_template.md`:

- **Jira Issue:** link para o ticket
- **O que foi feito:** bullet points do que foi implementado
- **Como testar:** passos concretos para validar
- **Regras Afetadas:** lista dos IDs criados ou modificados (ex: `REC-03`, `REC-07`)

---

## Restrições

- Não implemente nada antes do CHECKPOINT 1 ser aprovado
- Não marque o DoD como completo sem verificar cada item
- Se uma ambiguidade não foi respondida, pare e pergunte — não assuma
- `task-brief.yaml` não deve ser versionado (adicionar ao `.gitignore` se necessário)
