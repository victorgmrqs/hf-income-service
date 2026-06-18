---
name: docs-sync
description: >
  Sincroniza a documentação com as mudanças de código do branch atual
  (hf-income-service): aplica a tabela "mudou X → atualiza doc Y" (CHANGELOG,
  rules, openapi, INTEGRATIONS, ENTITIES, ADR) e sincroniza regras de negócio
  com o Notion via MCP. Chamada pela Fase 5 do /task ou avulsa.
  Uso: /docs-sync [TICKET-ID]
---

# Skill: /docs-sync (hf-income-service)

## Objetivo

Garantir a invariante: **código alterado = documentação alterada no mesmo branch**. Esta skill aplica as atualizações; a verificação (gate) é do `/code-review-task` + `docs-guard.sh` no CI.

## Etapa 1 — Detectar o que mudou

```bash
git diff development...HEAD --name-only        # ou git diff development --name-only + untracked (pré-commit)
git diff development...HEAD -- go.mod           # detectar mudança de dependências
```

## Etapa 2 — Tabela de sincronização (aplicar TODAS as linhas que casarem)

| Se o diff toca... | Atualizar |
|--------------------|-----------|
| Qualquer coisa em `internal/` | `CHANGELOG.md` → `[Unreleased]` (Added/Changed/Fixed), sufixo `(HF-XX)` |
| `go.mod` (deps) | `CHANGELOG.md` → seção **Dependencies** (`pacote: x.y.z → a.b.c`) |
| Entidade/coluna GORM nova | `ENTITIES.md` |
| Handler/rota/contrato REST | `openapi.yaml` + `.http` de evidência em `docs/http/<domain>/` |
| Chamada ao hf-transaction-service (httpclient) | `INTEGRATIONS.md` |
| Regra nova/modificada (`// DOMAIN-XX`) | `docs/rules/<DOMAIN>.md` (Regras) — **e Notion (Etapa 3)** |
| Decisão arquitetural tomada na task | Novo `docs/adr/ADR-XXX-*.md` |
| Fluxo/erro novo em domínio com FDD | FDD: fluxos (seção 4) e matriz de erros (seção 6) |

Regras de escrita: atualize **apenas** o que o diff justifica; datas absolutas (YYYY-MM-DD); IDs de regra/ticket sempre referenciados.

## Etapa 3 — Notion (regras de negócio — fonte de verdade)

Quando a Etapa 2 alterou regra de negócio:
1. Localize a página do domínio no workspace (https://www.notion.so/30ee65ad4ba98086809ed3a3f38ef45f) via MCP.
2. Atualize a regra (`updateConfluencePage`/Notion MCP equivalente), refletindo a mesma mudança — não reescreva a página inteira. Notion prevalece sobre o espelho `docs/rules/`.
3. Se a página ainda não existir: **não crie por conta própria**; registre como pendência no relatório.

## Etapa 4 — Relatório

```
## Docs Sync — <TICKET-ID>
Atualizados: CHANGELOG.md, docs/rules/REC.md, openapi.yaml, ...
Notion: [x] regra REC-07 atualizada | [ ] pendente: <motivo>
Sem mudança necessária: INTEGRATIONS.md, ENTITIES.md
```

## Restrições
- Não inventar conteúdo: cada atualização rastreável a uma linha do diff.
- Não criar páginas no Notion sem aprovação. Não tocar em código — só documentação.
