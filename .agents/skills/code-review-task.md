---
name: code-review-task
description: >
  Code review automatizado do branch atual contra development (hf-income-service).
  Valida aderência ao CLAUDE.md, executa gofmt/go vet/golangci-lint/go test,
  verifica cenários de teste obrigatórios, faz bug-hunt no diff, roda docs-guard
  e emite veredito APROVADO/REPROVADO. Chamada pela Fase 6.5 do /task ou avulsa.
  Uso: /code-review-task <TICKET-ID>   ex: /code-review-task HF-42
---

# Skill: /code-review-task (hf-income-service)

## Objetivo

Gate de qualidade antes do commit: revisar **apenas o diff do branch atual** contra `development` e emitir veredito objetivo. Não é refatoração — é verificação de conformidade.

## Etapa 1 — Coleta
1. Diff: `git diff development...HEAD` (+ `git status`). Pré-commit (fluxo /task): working tree.
2. Contexto: `CLAUDE.md`, `docs/workflow.md`, `docs/observability.md` e o FDD do domínio (matriz de erros seção 6).
3. Ticket: `task-brief.yaml` na raiz, ou via MCP Atlassian.

## Etapa 2 — Checklist de styleguide (análise do diff)

| # | Verificação |
|---|-------------|
| 1 | Dependência estrita `Handler → UseCase → Repository (interface) → DB`; use case não importa GORM nem `gin` |
| 2 | Valores monetários em `decimal.Decimal` — nunca `float64` |
| 3 | Toda variável de erro que implementa regra tem `// DOMAIN-XX` no ponto de aplicação |
| 4 | Respostas via `pkg/response` (`Success`/`Error`); códigos de erro batem com a matriz do FDD |
| 5 | Use case novo loga INFO entrada/saída; violação de regra incrementa `BusinessErrorsTotal{domain, rule_id}` |
| 6 | `trace_id` presente em logs de erro/warn; nenhum dado sensível logado (senha/token/key/conn string/body) |
| 7 | UUID v4 em `BeforeCreate`; soft delete via `gorm.DeletedAt`; query filtra dono quando aplicável |
| 8 | Sem segredo hardcoded, `.env` versionado ou chave em fixture |
| 9 | Nomenclatura/estrutura consistente com o vizinho (`internal/{entity,repository,usecase/<dom>,handler/<dom>}`) |

## Etapa 3 — Verificações executáveis (capturar resultado real)

```bash
gofmt -l .          # qualquer arquivo listado = não formatado = bloqueador
go vet ./...
golangci-lint run
go test ./...
```
Erro em qualquer um = **bloqueador**.

## Etapa 4 — Cenários de teste obrigatórios
Compare os `_test.go` com os "Cenários de Teste Obrigatórios" do ticket/FDD:
- `Test<UseCase>_Success` presente?
- Um teste por linha da matriz de erros do FDD seção 6?
- `Test<UseCase>_UpstreamError` quando o use case chama o hf-transaction-service?
- Repositório tocado tem teste de integração (testcontainers)?

Cenário obrigatório ausente = **bloqueador**. Teste trivial/sem asserção significativa conta como ausente.

## Etapa 4.5 — Docs-guard

Execute `scripts/docs-guard.sh` **e** verifique semanticamente:

| Se o diff contém... | Exigir no mesmo diff | Severidade |
|----------------------|----------------------|------------|
| Mudança em `internal/` | `CHANGELOG.md` `[Unreleased]` com `(HF-XX)` | Bloqueador |
| Mudança de dependência em `go.mod` | `CHANGELOG.md` seção Dependencies | Bloqueador |
| Entidade/coluna nova (GORM) | `ENTITIES.md` atualizado | Bloqueador |
| Mudança de endpoint/contrato REST | `openapi.yaml` + `.http` de evidência | Bloqueador |
| Contrato com hf-transaction-service | `INTEGRATIONS.md` | Bloqueador |
| Comentário `// DOMAIN-XX` novo | Regra existe em `docs/rules/<DOMAIN>.md` (e Notion) | Bloqueador |
| Use case sem teste no mesmo diff | `*_test.go` correspondente | Bloqueador |
| Decisão arquitetural | `docs/adr/ADR-XXX-*.md` | Recomendação |

## Etapa 4.6 — Bug-hunt no diff (procurar bugs, não estilo)

| # | Padrão de bug | Onde olhar |
|---|---------------|------------|
| 1 | Erro ignorado (`_ =`) ou não propagado; `err` engolido | retornos de repo/usecase/httpclient |
| 2 | Aritmética monetária com perda de precisão; comparação de `decimal` com `==` em vez de `.Equal` | cálculos de saldo/teto/meta |
| 3 | Competência (mês/ano) mal parseada ou comparada; borda de virada de ano | filtros por competência |
| 4 | Query sem filtro de soft delete / sem filtro de dono → vaza dado | repositórios |
| 5 | `nil` desreferenciado (ponteiro de struct/optional não checado) | mapeamento de DTO/resposta |
| 6 | Falha do httpclient (hf-transaction-service) não tratada / sem timeout / sem trace | chamadas cross-service |
| 7 | Race em goroutine; contexto não propagado | handlers concorrentes |
| 8 | Loop com índice/limite errado em paginação ou agregação | listagens/somatórios |

Bug provável com impacto real = **bloqueador**; suspeita/risco teórico = recomendação com justificativa.

## Etapa 4.8 — Review nativo de segurança
**`/security-review` (built-in) é OBRIGATÓRIO** quando o diff toca: autenticação/autorização, o httpclient para o hf-transaction-service, configuração de credenciais/env, ou exposição de dados em response/log. Achados de severidade alta = bloqueador.

## Etapa 5 — Veredito

```
## Code Review — <TICKET-ID>

**Veredito: APROVADO | REPROVADO**

### Bloqueadores
- [arquivo:linha] descrição + regra violada (ex: REC-03, gofmt, teste ausente)

### Recomendações
- [arquivo:linha] sugestão

### Execuções
- gofmt: ✅/❌ · go vet: ✅/❌ · golangci-lint: ✅/❌ · go test: ✅ N / ❌

### Cenários obrigatórios
- [x] Test..._Success  · [ ] Test..._<erro>  ← AUSENTE

### Docs-guard
- CHANGELOG: ✅/❌ · rules: ✅/—/❌ · openapi: ✅/—/❌ · INTEGRATIONS: ✅/—/❌ · ENTITIES: ✅/—/❌

### Bug-hunt
- N achados (por severidade) ou "nenhum padrão da Etapa 4.6 encontrado"
- /security-review nativo: executado ✅ (achados: N) / não exigido

### Checklist do PR (para a Fase 7 do /task)
- Itens verificados para pré-marcar: [lista]
```

## Restrições
- Não corrija código nesta skill; não aprove com bloqueador aberto; revise só o diff do branch.
