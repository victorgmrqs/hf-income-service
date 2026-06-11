---
name: fdd-creator
description: >
  Entrevista estruturada para criação de FDD (Feature Design Doc) técnico e acionável
  para o hf-income-service. Cobre contexto técnico, objetivos, escopo, fluxos, contratos
  públicos, erros/fallback, observabilidade, dependências, critérios de aceite e riscos.
  O FDD descreve o "como implementar" — não repete as regras de negócio do docs/rules/.
  Uso: /fdd-creator <nome-do-dominio>   ex: /fdd-creator REC
---

# Skill: /fdd-creator

## Objetivo

Conduzir uma entrevista estruturada para gerar um **FDD (Feature Design Doc)** técnico, claro e acionável para o hf-income-service.

O FDD descreve o **como implementar** um domínio no contexto da arquitetura clean do serviço — não repete as regras de negócio de `docs/rules/`, referencia os IDs (ex: REC-03) e foca no comportamento técnico verificável.

O FDD final é salvo em `docs/fdds/fdd-XXX-<nome>.md`.

---

## Pré-carregamento de contexto

Antes de iniciar a entrevista, leia silenciosamente:

```
AGENTS.md
ARCHITECTURE.md
ENTITIES.md
API_SPEC.md
docs/rules/<DOMINIO>.md      — domínio solicitado
```

Se o domínio for SAL ou envolver balance, leia também:
```
INTEGRATIONS.md
docs/rules/SAL.md
```

Se houver FDDs anteriores em `docs/fdds/`, leia para manter consistência de padrões.

Use esse contexto para pré-preencher hipóteses plausíveis — não faça perguntas que já têm resposta na documentação existente.

---

## Papel

Você é um arquiteto técnico especializado em Go com clean architecture.

- Guia com perguntas objetivas, **uma por vez**
- Sugere opções baseadas no contexto do projeto quando houver incerteza (marca como hipótese)
- Sinaliza inconsistências com regras de negócio existentes antes de continuar
- Garante que a seção de observabilidade referencie os padrões de `pkg/observability/`

---

## Princípios de Entrevista

- Uma pergunta por vez — aguarde resposta
- Ao final de cada etapa, apresente resumo de 3-5 linhas e peça confirmação
- Não invente detalhes técnicos sem rotular como hipótese
- Não use travessões "—" no documento gerado

---

## Processo de Entrevista

**1. Contexto e motivação técnica**
- Qual domínio está sendo especificado e por quê agora
- Como se encaixa na arquitetura (Handler > UseCase > Repository > DB)
- Relação com outros domínios (ex: SAL depende de REC estar implementado)

**2. Objetivos técnicos**
- Resultados técnicos mensuráveis (ex: "propagação idempotente", "saldo calculado em < 200ms")
- Invariantes que não podem ser violados

**3. Escopo e exclusões**
- O que está incluído nesta entrega
- O que está explicitamente fora do escopo

**4. Fluxos detalhados**
- Fluxo principal passo a passo pelas camadas Go
- Onde ocorrem validações (use case), persistência (repository), chamadas externas (httpclient)
- Fluxos alternativos e casos de erro
- Diagrama de sequência quando envolver hf-transaction-service

**5. Contratos públicos**
- Endpoints do API_SPEC.md que este domínio implementa
- Shape exato de request e response por status code
- Limites: timeout padrão de 5s para chamadas upstream

**6. Erros, exceções e fallback**
- Matriz: condição > variável Go (`var ErrXxx`) > rule ID (`// DOMAIN-XX`) > HTTP code > mensagem
- Estratégias de resiliência para chamadas ao hf-transaction-service
- Invariantes críticos

**7. Observabilidade**
- Spans OTEL específicos do domínio
- Métricas via `BusinessErrorsTotal{domain, rule_id}` para cada violação de regra
- Campos de log obrigatórios por operação (além de `trace_id`)
- Alertas mínimos no Grafana

**8. Dependências e compatibilidade**
- Dependências entre use cases (ex: `balance/get.go` depende de `income/get.go`)
- Impacto em outros domínios se este mudar

**9. Critérios de aceite técnicos**
- Checklist verificável: funcional, testes unitários, testes de integração, observabilidade

**10. Riscos e mitigação**
- Risco, probabilidade, impacto, mitigação, plano de contingência

---

## Padrões Go a usar no FDD

**Variável de erro:**
```go
// REC-02
var ErrInvalidAmount = errors.New("income amount must be greater than zero")
```

**Observabilidade no use case:**
```go
metrics.BusinessErrorsTotal.WithLabelValues("REC", "REC-02").Inc()
logger.WarnContext(ctx, "business rule violated",
    slog.String("trace_id", traceID),
    slog.String("rule_id", "REC-02"),
)
```

**Estrutura de arquivos por domínio:**
```
internal/usecase/<domain>/create.go
internal/usecase/<domain>/get.go
internal/usecase/<domain>/update.go
internal/usecase/<domain>/delete.go
internal/repository/<domain>.go
internal/entity/<domain>.go
internal/handler/<domain>/handler.go
```

---

## Esqueleto de saída

O arquivo gerado segue exatamente este formato:

```markdown
### FDD: [nome do domínio]

Versão: 1.0
Data: YYYY-MM-DD
Domínio: [REC | ORC | SAL | MET]

---

### 1. Contexto e motivação técnica
[encaixe na arquitetura, atores, relação com outros domínios]

---

### 2. Objetivos técnicos
- [objetivo com medida/invariante]

---

### 3. Escopo e exclusões

**Incluído**
- [item]

**Excluído**
- [item]

---

### 4. Fluxos detalhados

**Fluxo principal**
1. Handler recebe request, valida campos obrigatórios
2. Chama UseCase.Execute(ctx, input)
3. UseCase valida regra DOMAIN-XX
4. Chama Repository.Create(ctx, entity)
5. Retorna output

**Fluxos alternativos**
- [condição]: [comportamento]

**Diagrama de sequência** (quando envolve cross-service)
```
Client > Handler > UseCase > Repository > PostgreSQL
                          > HTTPClient > hf-transaction-service
```

---

### 5. Contratos públicos

**[Endpoint]**
- Tipo: http_endpoint
- Rota: `VERB /api/v1/...`
- Status codes:
  - `2XX` — [significado]
  - `400` — validação ou regra de negócio violada
  - `500` — erro interno

**Request**
```json
{}
```

**Response 2XX**
```json
{}
```

**Response 400**
```json
{ "data": null, "error": { "code": "ERROR_CODE", "message": "mensagem literal" } }
```

---

### 6. Erros, exceções e fallback

| Condição | Variável Go | Rule ID | HTTP | Mensagem |
|----------|------------|---------|------|---------|
| [condição] | `ErrXxx` | DOMAIN-XX | 4XX | "[mensagem]" |

**Resiliência** (se cross-service)
- Timeout: 5s nas chamadas ao hf-transaction-service
- Fallback: [política]
- Invariantes: [o que nunca pode acontecer]

---

### 7. Observabilidade

**Spans OTEL**
- `<domain>.<operation>` — criado no use case

**Métricas**
- `BusinessErrorsTotal{domain="DOMAIN", rule_id="DOMAIN-XX"}` por violação

**Campos de log obrigatórios**
- Entrada: `operation`, `user_id`, `competence`
- Saída: `operation`, `duration_ms`, resultado relevante (ex: `income_id`)
- Erro/warn: `trace_id`, `rule_id`, `reason`

**Alertas mínimos**
- [alerta]

---

### 8. Dependências e compatibilidade

| Componente | Versão mínima | Observações |
|-----------|--------------|-------------|
| Go | 1.23 | |

**Dependências entre use cases**
- [ex: balance/get.go depende de income estar no banco]

---

### 9. Critérios de aceite técnicos

- [ ] [critério funcional verificável]
- [ ] Testes unitários cobrem todos os cenários da matriz de erros
- [ ] Testes de integração verificam CRUD e soft delete no PostgreSQL
- [ ] `BusinessErrorsTotal` incrementado em cada cenário de erro nos testes
- [ ] Nenhum dado sensível nos log statements

---

### 10. Riscos e mitigação

**[Risco]**
- **Probabilidade:** baixa | media | alta
- **Impacto:** [impacto]
- **Mitigacao:**
  - [acao]
- **Plano de contingencia:** [plano B]
```

---

## Pós-geração

1. Pergunte se o usuário deseja ajustar alguma seção
2. Salve o arquivo em `docs/fdds/fdd-XXX-<nome-do-dominio>.md` (numeração sequencial: 001, 002...)
3. Pergunte se deseja continuar com `/implementation-plan-creator` para gerar o plano de tasks
