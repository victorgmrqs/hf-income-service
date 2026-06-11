---
name: implementation-plan-creator
description: >
  Analisa os FDDs em docs/fdds/ e gera um plano de implementação ordenado e faseado
  para o hf-income-service. Produz tasks numeradas (T01, T02...) com estimativas
  P/M/G/GG, dependências, critérios de aceite e pontos de checkpoint.
  Uso: /implementation-plan-creator
---

# Skill: /implementation-plan-creator

## Objetivo

Ler todos os FDDs em `docs/fdds/` e gerar um **plano de implementação faseado** com tasks ordenadas por dependências técnicas, estimativas de esforço e critérios de aceite verificáveis.

O plano é salvo em `docs/implementation-plan.md`.

---

## Pré-carregamento de contexto

Leia silenciosamente antes de iniciar:

```
AGENTS.md
ARCHITECTURE.md
ENTITIES.md
docs/fdds/*.md        — todos os FDDs existentes
```

Se houver FDDs de domínios com dependência entre si (ex: SAL depende de REC), identifique as dependências antes de ordenar as tasks.

---

## Regras de geração

1. **Testes são parte da task, não tasks separadas** — cada T0X inclui testes unitários e de integração no critério de aceite
2. **Observabilidade é requisito por task** — não é uma fase final; cada task deve incluir spans, métricas e logs
3. **Cada task deve ser independentemente verificável** — critério de aceite binário: passou ou não
4. **Infraestrutura antes de feature** — migração/schema antes do CRUD que depende dela
5. **Tasks mais simples primeiro dentro de uma fase** — reduz risco de bloqueio por dependência
6. **Cross-service tasks ficam na última fase** — SAL (balance) depende de tudo estar funcionando
7. **Checkpoints entre fases** — após cada fase há um checkpoint de revisão antes de continuar
8. **Estimativas são em dias de trabalho individual:** P = 0.5d, M = 1d, G = 2d, GG = 3-5d
9. **Uma task por arquivo que vai ser criado/modificado** — evitar tasks "crie tudo do domínio X"
10. **Riscos técnicos elevam o tamanho da estimativa** — se o FDD tem risco alto, aumente a estimativa

---

## Tamanhos de estimativa

| Tamanho | Esforço | Quando usar |
|---------|---------|-------------|
| P | ~0.5 dia | Arquivo único, mudança bem definida, sem dependência complexa |
| M | ~1 dia | 2-4 arquivos, validação simples, caminho bem documentado |
| G | ~2 dias | Cross-layer (handler+usecase+repo+tests), regras não triviais |
| GG | 3-5 dias | Cross-service, decisão arquitetural, impacto em múltiplos domínios |

---

## Faseamento padrão para hf-income-service

**Fase 0 — Infraestrutura base**
- Entidades GORM (`internal/entity/`)
- Migrações (documentadas em ENTITIES.md)
- Interfaces de repositório (`internal/repository/interfaces.go`)
- pkg/httpclient para hf-transaction-service (se ainda não existir)

**Fase 1 — Domínio REC (Receitas)**
- Repositório income
- Use cases: create, get (por user/competence), update, delete
- Handler + rotas
- Testes unitários e de integração

**Fase 2 — Domínio ORC (Orçamento Global)**
- Repositório global_budget
- Use cases: create/update ceiling, auto-adjust logic
- Handler + rotas
- Testes unitários e de integração

**Fase 3 — Domínio MET (Metas de Redução)**
- Repositório reduction_goal
- Use cases: create, get, update, delete, calculate progress
- Handler + rotas
- Testes unitários e de integração

**Fase 4 — Domínio SAL (Saldo Mensal)**
- HTTP client para hf-transaction-service (GET expenses/totals, GET accounts-payable)
- Use case get balance (computed)
- Handler + rota
- Testes unitários (mock do HTTP client)

**Fase 5 — Integração e validação**
- Testes e2e com servidor real e banco PostgreSQL
- Smoke tests dos endpoints críticos
- Validação de observabilidade (spans, métricas, logs)

---

## Processo de análise

**1. Extração de tasks por FDD**
Para cada FDD lido, extraia:
- Endpoints da seção 5 (Contratos públicos)
- Entidades da seção 4 (Fluxos — o que vai ao banco)
- Erros da seção 6 (cada `var ErrXxx` é um item de teste)
- Dependências da seção 8

**2. Identificação de dependências entre tasks**
Marque qual T0X precisa estar concluído antes de outra poder começar.

**3. Agrupamento em fases**
Use o faseamento padrão acima. Ajuste se os FDDs indicarem dependências diferentes.

**4. Estimativa de esforço**
Para cada task, estime com base nos critérios da tabela.

**5. Identificação de riscos técnicos**
Tasks com risco alto (do FDD) recebem aviso.

---

## Esqueleto de saída

```markdown
# Plano de Implementação — hf-income-service

Data: YYYY-MM-DD
FDDs considerados: [lista]
Estimativa total: X dias

---

## Resumo de fases

| Fase | Domínio | Tasks | Estimativa |
|------|---------|-------|-----------|
| 0 | Infraestrutura | T01-T03 | Xd |
| 1 | REC | T04-T09 | Xd |
| ... | ... | ... | ... |

---

## Fase 0 — Infraestrutura base

### T01 — Entidades GORM e migrações
**Estimativa:** P | M | G | GG
**Depende de:** nada
**Arquivos:**
- `internal/entity/income.go`
- `internal/entity/global_budget.go`
- `internal/entity/reduction_goal.go`
- `ENTITIES.md` (atualizar)

**Critérios de aceite:**
- [ ] Entidades compilam sem erro
- [ ] `gorm.AutoMigrate` cria as tabelas no PostgreSQL
- [ ] UUIDs gerados em BeforeCreate
- [ ] Soft delete configurado com `gorm.DeletedAt`

**Risco:** baixo | medio | alto

---

> [checkpoint após Fase 0]
> Antes de continuar para Fase 1, verificar:
> - Banco sobe corretamente com AutoMigrate
> - Estrutura de pastas está correta

---

## Riscos consolidados

| Task | Risco | Motivo | Mitigacao |
|------|-------|--------|-----------|
| T0X | medio | [motivo do FDD] | [acao] |

---

## Proximos passos

Apos aprovacao deste plano:
1. Criar tickets Jira para cada task (ou agrupar P/M em um unico ticket)
2. Iniciar pela Fase 0
3. Usar `/task <TICKET-ID>` para executar cada task com o workflow completo
```

---

## Pós-geração

1. Apresente o resumo de fases ao usuário e pergunte se há ajustes
2. Pergunte se deseja criar os tickets Jira automaticamente via Atlassian MCP
3. Salve o arquivo em `docs/implementation-plan.md`
