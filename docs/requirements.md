# Requisitos — hf-income-service

> Fonte de verdade: Notion. Este arquivo é espelho local versionado.
> Notion (workspace): https://www.notion.so/30ee65ad4ba98086809ed3a3f38ef45f
> Em caso de divergência, o Notion prevalece.

---

## Requisitos Funcionais (RF)

Os requisitos funcionais estão detalhados por domínio em `docs/rules/`. Esta seção lista os casos de uso de cada domínio com prioridade e rastreabilidade.

**Legenda de prioridade:**
- `MUST` — obrigatório para o MVP
- `SHOULD` — importante, mas não bloqueia o MVP
- `COULD` — desejável em versão futura

---

### RF-REC — Receitas

> Regras de negócio: [docs/rules/REC.md](rules/REC.md)

| ID | Caso de Uso | Prioridade | Regras |
|----|-------------|------------|--------|
| RF-REC-01 | Registrar uma nova receita (description, amount, date, competence, type, user_id) | MUST | REC-01, REC-02 |
| RF-REC-02 | Listar receitas de um usuário por competência | MUST | REC-01 |
| RF-REC-03 | Editar receita existente sem alterar histórico de competências anteriores | MUST | REC-03 |
| RF-REC-04 | Excluir receita (soft delete) | MUST | REC-01 |
| RF-REC-05 | Marcar receita como recorrente | SHOULD | REC-03, REC-04 |
| RF-REC-06 | Propagar receitas recorrentes para a próxima competência | SHOULD | REC-04 |
| RF-REC-07 | Calcular total de receitas por competência e usuário | MUST | REC-05, REC-06 |

---

### RF-ORC — Orçamento Global (Teto de Gastos)

> Regras de negócio: [docs/rules/ORC.md](rules/ORC.md)

| ID | Caso de Uso | Prioridade | Regras |
|----|-------------|------------|--------|
| RF-ORC-01 | Definir teto global de gastos para uma competência | MUST | ORC-01, ORC-02 |
| RF-ORC-02 | Consultar teto vigente para uma competência | MUST | ORC-01 |
| RF-ORC-03 | Ajustar automaticamente o teto com base nos gastos do mês anterior | SHOULD | ORC-03, ORC-04, ORC-05 |
| RF-ORC-04 | Registrar flag `auto_adjusted` quando teto foi ajustado automaticamente | SHOULD | ORC-04 |
| RF-ORC-05 | Editar teto manualmente, sobrescrevendo auto-ajuste | MUST | ORC-06 |

---

### RF-SAL — Saldo Mensal

> Regras de negócio: [docs/rules/SAL.md](rules/SAL.md)
> Integrações: [INTEGRATIONS.md](../INTEGRATIONS.md)

| ID | Caso de Uso | Prioridade | Regras |
|----|-------------|------------|--------|
| RF-SAL-01 | Calcular saldo real do mês (receitas - despesas realizadas) | MUST | SAL-01, SAL-02 |
| RF-SAL-02 | Calcular saldo projetado (saldo real - despesas a vencer) | MUST | SAL-03 |
| RF-SAL-03 | Indicar se o saldo está dentro do teto global | SHOULD | SAL-04 |
| RF-SAL-04 | Retornar breakdown: receita_total, despesa_realizada, despesa_comprometida, saldo_real, saldo_projetado | MUST | SAL-05 |

---

### RF-MET — Metas de Redução de Gastos

> Regras de negócio: [docs/rules/MET.md](rules/MET.md)

| ID | Caso de Uso | Prioridade | Regras |
|----|-------------|------------|--------|
| RF-MET-01 | Criar meta de redução para uma categoria e competência | MUST | MET-04 |
| RF-MET-02 | Consultar progresso da meta (gasto atual vs. limite definido) | MUST | MET-05 |
| RF-MET-03 | Editar meta existente | MUST | MET-04 |
| RF-MET-04 | Excluir meta (soft delete) | MUST | MET-04 |
| RF-MET-05 | Listar metas de um usuário por competência | MUST | MET-04 |
| RF-MET-06 | Calcular percentual de atingimento da meta | SHOULD | MET-06, MET-07 |

---

## Requisitos Não Funcionais (RNF)

Os RNFs são transversais a todos os domínios. Cada item tem target mensurável e método de verificação.

---

### RNF-PER — Desempenho

| ID | Requisito | Target | Método de Verificação |
|----|-----------|--------|----------------------|
| RNF-PER-01 | Latência p95 para endpoints de leitura (GET) | < 300ms | Prometheus histogram `http_request_duration_seconds{quantile="0.95"}` |
| RNF-PER-02 | Latência p95 para endpoints de escrita (POST/PUT/DELETE) | < 500ms | Prometheus histogram |
| RNF-PER-03 | Latência p95 para `/balance` (envolve chamada ao hf-transaction-service) | < 1s | Prometheus histogram + trace no Grafana Tempo |
| RNF-PER-04 | Tempo de startup da aplicação | < 5s | Medido no log de inicialização |

---

### RNF-DIS — Disponibilidade e Confiabilidade

| ID | Requisito | Target | Método de Verificação |
|----|-----------|--------|----------------------|
| RNF-DIS-01 | Disponibilidade mensal do serviço | >= 99% | Uptime monitor no Grafana Cloud |
| RNF-DIS-02 | Falha no hf-transaction-service não derruba o serviço inteiro | Degradação parcial: SAL retorna erro, demais endpoints funcionam | Teste de integração com mock de falha |
| RNF-DIS-03 | Timeout nas chamadas ao hf-transaction-service | 5s (hard limit) | Configuração no `pkg/httpclient/` |

---

### RNF-SEG — Segurança

| ID | Requisito | Prioridade | Observação |
|----|-----------|------------|-----------|
| RNF-SEG-01 | Autenticação JWT em todos os endpoints | MUST (próxima iteração) | Não implementado no MVP; auth será adicionada como caminho separado |
| RNF-SEG-02 | Nenhum dado sensível em logs (passwords, tokens, API keys, connection strings, bodies) | MUST | Enforced via convenção em CLAUDE.md e code review |
| RNF-SEG-03 | Comunicação HTTPS em produção | MUST | Gerenciado no nível do ingress (homelab-gitops) |
| RNF-SEG-04 | Variáveis sensíveis via env vars, nunca hardcoded | MUST | `.env.example` sem valores reais; `.env` no `.gitignore` |

---

### RNF-OBS — Observabilidade

> Detalhamento completo: [docs/observability.md](observability.md)

| ID | Requisito | Target | Método de Verificação |
|----|-----------|--------|----------------------|
| RNF-OBS-01 | Logs estruturados (JSON) em produção com `trace_id` em cada linha de erro/warn | 100% dos log statements de erro e warn | Code review + teste manual |
| RNF-OBS-02 | Métricas Prometheus expostas em `/metrics` (porta 9090) | Sempre ativas | `curl :9090/metrics` retorna 200 |
| RNF-OBS-03 | Traces OTEL enviados ao Grafana Tempo em produção | Sampling rate configurável via `OTEL_SAMPLING_RATIO` (default 1.0) | Grafana Explore → Tempo |
| RNF-OBS-04 | Violações de regra de negócio incrementam `business_errors_total{domain, rule_id}` | 100% dos erros de domínio | Testes de integração verificam o contador |

---

### RNF-MAN — Manutenibilidade

| ID | Requisito | Target | Método de Verificação |
|----|-----------|--------|----------------------|
| RNF-MAN-01 | Cobertura de testes unitários nos use cases | >= 80% | `go test -cover ./internal/usecase/...` |
| RNF-MAN-02 | Cobertura de testes de integração nos repositórios | Todos os métodos públicos cobertos | `go test -cover ./internal/repository/...` |
| RNF-MAN-03 | Build sem warnings | 0 warnings | `go build ./...` no CI |
| RNF-MAN-04 | Regras de negócio documentadas antes de implementadas | 100% | DoD item: regra no Notion antes do código |

---

### RNF-ESC — Escalabilidade

| ID | Requisito | Observação |
|----|-----------|-----------|
| RNF-ESC-01 | O serviço é stateless — sem estado em memória entre requests | Permite múltiplas réplicas sem coordenação |
| RNF-ESC-02 | Volume esperado: < 10 usuários, < 1.000 registros/mês por usuário | Serviço pessoal; escalabilidade horizontal não é prioridade no MVP |

---

## Rastreabilidade

| RF | Domínio | Regras de Negócio | RNF Relacionados |
|----|---------|-------------------|-----------------|
| RF-REC | REC | REC-01 a REC-06 | RNF-PER-01/02, RNF-MAN-01/02 |
| RF-ORC | ORC | ORC-01 a ORC-07 | RNF-PER-01/02, RNF-MAN-01/02 |
| RF-SAL | SAL | SAL-01 a SAL-05 | RNF-PER-03, RNF-DIS-02/03 |
| RF-MET | MET | MET-04 a MET-07 | RNF-PER-01/02, RNF-MAN-01/02 |
