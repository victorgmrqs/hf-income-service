# Arquitetura do Sistema — hf-income-service

## Stack Tecnológica

| Componente | Tecnologia |
|---|---|
| Linguagem | Go 1.23+ |
| Framework HTTP | Gin |
| ORM | GORM |
| Banco de dados | PostgreSQL 16 |
| UUID | google/uuid |
| Decimal | shopspring/decimal |
| Config | spf13/viper |
| Testes | testcontainers-go (integração) |

## Estrutura de Pastas

```
hf-income-service/
├── cmd/
│   └── server/
│       └── main.go              # Entrypoint — wiring de dependências
├── config/
│   └── config.go                # Carregamento de variáveis via Viper
├── internal/
│   ├── entity/
│   │   ├── income.go            # Modelo Income (REC)
│   │   ├── global_budget.go     # Modelo GlobalBudget (ORC)
│   │   └── reduction_goal.go    # Modelo ReductionGoal (MET)
│   ├── repository/
│   │   ├── interfaces.go        # Interfaces de repositório
│   │   ├── income.go
│   │   ├── global_budget.go
│   │   └── reduction_goal.go
│   ├── usecase/
│   │   ├── income/
│   │   │   ├── create.go
│   │   │   ├── get.go
│   │   │   ├── update.go
│   │   │   ├── delete.go
│   │   │   └── propagate.go     # Propagação de recorrências mensais
│   │   ├── global_budget/
│   │   │   ├── create.go
│   │   │   ├── get.go
│   │   │   ├── update.go
│   │   │   └── auto_adjust.go   # ORC-03/04/05
│   │   ├── reduction_goal/
│   │   │   ├── create.go
│   │   │   ├── get.go
│   │   │   ├── update.go
│   │   │   ├── delete.go
│   │   │   └── comparison.go    # MET-05/07
│   │   └── balance/
│   │       └── get.go           # SAL-01/02/03 — agrega income + chamada ao hf-transaction-service
│   └── handler/
│       ├── routes.go            # Registro de todas as rotas sob /api/v1/
│       ├── income/
│       ├── global_budget/
│       ├── reduction_goal/
│       └── balance/
├── pkg/
│   ├── database/
│   │   └── postgres.go          # Conexão GORM + auto-migrate
│   ├── response/
│   │   └── response.go          # response.Success / response.Error
│   └── httpclient/
│       └── transaction_client.go # Cliente HTTP para hf-transaction-service
├── openapi.yaml
├── CLAUDE.md
├── RULES.md
├── ARCHITECTURE.md
├── ENTITIES.md
├── API_SPEC.md
├── .env.example
├── Dockerfile
└── go.mod
```

## Fluxo de Requisição

```
Request → Gin Router → Handler → UseCase.Execute(input) → Repository.Method() → PostgreSQL
                                      ↓ (apenas balance)
                               httpclient.GetExpenseTotals()
                               httpclient.GetAccountsPayable()
                                      ↓
                               hf-transaction-service API
```

## Regra de Dependência

```
Handler → UseCase interface → Repository interface → GORM/DB
Handler → UseCase interface → HTTPClient interface → hf-transaction-service
```

Nenhuma camada interna importa camadas externas. Interfaces permitem mock em testes.

## Comunicação entre Serviços

| Chamada | Destino | Uso |
|---|---|---|
| `GET /expenses/totals?user_id=&competence=` | hf-transaction-service | SAL-01: total de despesas do mês |
| `GET /accounts-payable?user_id=&status=PENDING` | hf-transaction-service | SAL-02: despesas comprometidas |
| `GET /budgets/status?user_id=&competence=` | hf-transaction-service | ORC-06: soma de orçamentos por categoria |

## Convenções

- UUID v4 como PK gerado em `BeforeCreate`
- Soft delete via `gorm.DeletedAt` em todas as entidades
- Formato de competência: `YYYY-MM` (validado por regex)
- Valores monetários: `decimal.Decimal` (shopspring) — nunca `float64`
- Todas as respostas seguem o padrão `{ "data": ..., "error": null }` ou `{ "data": null, "error": { "code": "...", "message": "..." } }`

## Variáveis de Ambiente

| Variável | Descrição | Default |
|---|---|---|
| `APP_PORT` | Porta HTTP | `8081` |
| `APP_ENV` | Ambiente (`development`/`production`) | `development` |
| `DB_HOST` | Host PostgreSQL | `localhost` |
| `DB_PORT` | Porta PostgreSQL | `5432` |
| `DB_USER` | Usuário do banco | — |
| `DB_PASSWORD` | Senha do banco | — |
| `DB_NAME` | Nome do banco | `hf-income` |
| `TRANSACTION_SERVICE_URL` | URL base do hf-transaction-service | `http://localhost:8080` |
