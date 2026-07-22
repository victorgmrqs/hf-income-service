# Observabilidade — hf-income-service

## Visão geral

Este serviço implementa os três pilares de observabilidade:

```
Logs    → slog (JSON) → stdout → Grafana Alloy → Loki    → Grafana
Métricas → prometheus/client_golang → /metrics  → Alloy   → Prometheus → Grafana
Traces  → OpenTelemetry SDK → OTLP HTTP         → Grafana Tempo / Jaeger → Grafana
```

A instrumentação vive em `pkg/observability/` e é inicializada em `cmd/server/main.go` antes do servidor HTTP subir.

---

## Dependências a adicionar ao go.mod

```bash
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp
go get go.opentelemetry.io/otel/sdk
go get go.opentelemetry.io/otel/propagation
go get go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promauto
go get github.com/prometheus/client_golang/prometheus/promhttp
```

---

## Variáveis de ambiente

| Variável | Obrigatória | Padrão | Descrição |
|----------|------------|--------|-----------|
| `APP_ENV` | Sim | `development` | `development` → logs text; `production` → logs JSON |
| `OTEL_SERVICE_NAME` | Não | `hf-income-service` | Nome do serviço nos traces |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Prod | — | URL do gateway OTLP da Grafana Cloud |
| `OTEL_EXPORTER_OTLP_HEADERS` | Prod | — | `Authorization=Basic <base64(instanceId:token)>` |
| `OTEL_SAMPLING_RATIO` | Não | `1.0` | `1.0` = 100%, `0.1` = 10% |
| `APP_METRICS_PORT` | Não | `9090` | Porta onde `/metrics` é exposto |

> `OTEL_EXPORTER_OTLP_ENDPOINT` e `OTEL_EXPORTER_OTLP_HEADERS` são lidos automaticamente pelo SDK OTEL — não precisam de código adicional.

---

## Configuração no cmd/server/main.go

```go
func main() {
    cfg := config.Load()
    logger := observability.NewLogger(cfg.AppEnv)

    // Tracing — desabilitado em development se OTEL_EXPORTER_OTLP_ENDPOINT não estiver setado
    _, shutdownTracing, err := observability.NewTracerProvider(context.Background())
    if err != nil {
        logger.Warn("tracing disabled", slog.String("reason", err.Error()))
    } else {
        defer shutdownTracing()
    }

    // Métricas
    metrics := observability.NewServiceMetrics("hf_income")

    // HTTP server
    router := gin.New()
    router.Use(observability.RequestMiddleware(logger, metrics))
    router.GET("/metrics", gin.WrapH(promhttp.Handler()))

    // ... demais rotas
}
```

---

## O que está instrumentado

### Automático (via RequestMiddleware)
- Latência de todas as rotas (`hf_income_http_request_duration_seconds`)
- Volume de requisições (`hf_income_http_requests_total`)
- Trace span por request com `trace_id` propagado nos logs

### Manual (a adicionar nos use cases)
- Violações de regras de negócio: `metrics.BusinessErrorsTotal.WithLabelValues("REC", "REC-02").Inc()`
- Eventos importantes: `logger.InfoContext(ctx, "income propagated", slog.Int("count", n))`

### Scheduler interno (`src/internal/scheduler`, HF-38)
Goroutine em background, sem request HTTP associado — não passa pelo `RequestMiddleware`, então não há `trace_id` de request; os logs de erro/warn seguem a mesma convenção usando o `trace_id` do contexto quando presente (vazio em execução de background pura).

| Momento | Nível | Campos |
|---------|-------|--------|
| Boot da goroutine | INFO | `tz`, `interval` |
| Início do ciclo mensal | INFO | `operation="scheduler.monthly_jobs"`, `competence` |
| Falha de job/usuário (propagate ou auto_adjust) | ERROR | `trace_id`, `competence` (e `user_id` no auto_adjust), `error` — não aborta o ciclo |
| Fim do ciclo mensal | INFO | `operation`, `duration_ms`, `competence`, `propagated`, `adjusted`, `failed` |
| `SCHEDULER_TZ` inválida | WARN | `tz`, `error` — cai para UTC, não derruba o boot |
| `SCHEDULER_ENABLED=false` | INFO | goroutine retorna sem disparar nenhum job |

---

## Política de dados sensíveis

### NUNCA logar
| Dado | Motivo |
|------|--------|
| Passwords, tokens, API keys | Credenciais de acesso |
| Strings de conexão com banco (`DB_DSN`) | Credenciais de infraestrutura |
| Headers de autorização (`Authorization`, `OTEL_EXPORTER_OTLP_HEADERS`) | Tokens de serviço |
| Conteúdo completo de request/response bodies | Potencial PII e dados financeiros |

### Logar como referência (nunca como valor absoluto em erros)
| Dado | Como logar |
|------|-----------|
| Valores monetários em contexto de erro | Use `income_id`, não o `amount` |
| `user_id` | UUID — seguro para logs |
| `competence` | YYYY-MM — seguro para logs |

### Campos obrigatórios em todo log statement
```go
slog.String("trace_id", traceID)   // sempre — correlaciona com traces no Grafana
```

---

## Níveis de log por camada

| Camada | Nível | O que logar |
|--------|-------|-------------|
| Handler | `INFO` | Request iniciada e concluída (método, rota, status, duration_ms, trace_id) |
| Handler | `ERROR` | Falha de parse de input — sem expor o valor recebido |
| UseCase | `INFO` | Operação iniciada com contexto (operation, user_id, competence) |
| UseCase | `WARN` | Regra de negócio violada (rule_id, motivo) |
| UseCase | `ERROR` | Falha inesperada não relacionada a regra de negócio |
| Repository | `ERROR` | Falha de query (operation, entity) — sem valores de campo |
| Repository | `WARN` | Not found esperado (quando relevante para diagnóstico) |
| HTTPClient | `INFO` | Chamada iniciada ao upstream (service, endpoint) |
| HTTPClient | `ERROR` | Falha no upstream (status_code, service) — sem response body |
| HTTPClient | `WARN` | Timeout ou retry |

---

## Grafana Alloy (coleta)

A configuração do Alloy (scraping de `/metrics` e coleta de logs do stdout) está no repositório de infraestrutura:

```
homelab-gitops/base/hf-income-service/alloy-config.river
```

O Alloy roda como sidecar no mesmo pod e:
- Scrapa `/metrics` a cada 15s → envia ao Prometheus (Grafana Cloud)
- Lê logs do stdout (JSON) → envia ao Loki (Grafana Cloud)
- Não requer configuração adicional na aplicação
