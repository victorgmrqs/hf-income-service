---
name: testing-guide
description: >
  Guia de testes permanente do hf-income-service.
  Consultado automaticamente ao escrever ou revisar testes.
  Define o que testar, em qual camada e como configurar cada tipo.
  Uso: /testing-guide
---

# Guia de Testes — hf-income-service

## Princípio central

Cada camada responde uma pergunta diferente. Não duplique cobertura entre camadas.

| Camada | Pergunta | Mocka o quê |
|--------|----------|-------------|
| **Unit** | A lógica está correta? | Interfaces de repositório + TransactionClient |
| **Integration** | O contrato com o banco está correto? | Nada — usa PostgreSQL real via testcontainers |

Não existe camada E2E neste serviço — os handlers são simples (parse HTTP → chamar use case). A cobertura de contrato HTTP fica no frontend ou em testes de contrato externos.

---

## O que testar (e o que não testar)

### Vale a pena testar
- Use cases com lógica de negócio: validações, cálculos, decisões condicionais
- Propagação de receitas recorrentes (REC-03, REC-04)
- Auto-ajuste do teto global (ORC-03, ORC-04, ORC-05)
- Cálculo de saldo: `balance_today`, `projected_balance`, `is_projected_negative` (SAL-01 a SAL-05)
- Comportamento de erro: campos inválidos, regras violadas, upstream indisponível
- Repositórios: CRUD real contra PostgreSQL, soft delete, filtros por `user_id`/`competence`

### Não vale a pena testar
- Handlers (apenas fazem parse e delegam — sem lógica)
- Getters/setters sem lógica
- GORM AutoMigrate (responsabilidade do framework)
- Configuração de variáveis de ambiente

---

## Testes unitários (use cases)

### Estrutura do arquivo
```
internal/usecase/<domain>/<operation>_test.go
```

### Setup padrão

```go
package income_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/google/uuid"
    "github.com/shopspring/decimal"
)

// Mock manual da interface de repositório
type mockIncomeRepository struct {
    createFn func(ctx context.Context, income *entity.Income) error
    // adicione outros métodos conforme a interface
}

func (m *mockIncomeRepository) Create(ctx context.Context, income *entity.Income) error {
    return m.createFn(ctx, income)
}

func TestIncomeUseCase_Create(t *testing.T) {
    t.Run("success", func(t *testing.T) {
        repo := &mockIncomeRepository{
            createFn: func(_ context.Context, _ *entity.Income) error { return nil },
        }
        uc := income.NewCreateUseCase(repo)

        output, err := uc.Execute(context.Background(), income.CreateInput{
            UserID:      uuid.New(),
            Description: "Salário",
            Amount:      decimal.NewFromFloat(7500),
            Type:        entity.IncomeTypeSalary,
            Competence:  "2026-06",
            Recurrent:   true,
        })

        assert.NoError(t, err)
        assert.NotEmpty(t, output.ID)
    })

    t.Run("invalid amount returns error", func(t *testing.T) {
        repo := &mockIncomeRepository{}
        uc := income.NewCreateUseCase(repo)

        _, err := uc.Execute(context.Background(), income.CreateInput{
            Amount: decimal.Zero, // REC-02
        })

        assert.ErrorIs(t, err, income.ErrInvalidAmount)
    })
}
```

### Mocks
- Escreva mocks manualmente implementando as interfaces de `internal/repository/interfaces.go`
- Um mock por test file — não compartilhe mocks entre pacotes de test
- Para `TransactionClient`: mock da interface `pkg/httpclient/interfaces.go`

---

## Testes de integração (repositórios)

### Estrutura do arquivo
```
internal/repository/<entity>_test.go
```

### Setup com testcontainers-go

```go
package repository_test

import (
    "context"
    "testing"

    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
    t.Helper()
    ctx := context.Background()

    pgContainer, err := testpostgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:16-alpine"),
        testpostgres.WithDatabase("testdb"),
        testpostgres.WithUsername("test"),
        testpostgres.WithPassword("test"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2),
        ),
    )
    if err != nil {
        t.Fatalf("failed to start postgres container: %v", err)
    }

    t.Cleanup(func() { pgContainer.Terminate(ctx) })

    connStr, _ := pgContainer.ConnectionString(ctx, "sslmode=disable")
    db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{})
    if err != nil {
        t.Fatalf("failed to connect to test db: %v", err)
    }

    // AutoMigrate apenas as entidades necessárias para o teste
    db.AutoMigrate(&entity.Income{})

    return db
}

func TestIncomeRepository_Create(t *testing.T) {
    db := setupTestDB(t)
    repo := repository.NewIncomeRepository(db)

    income := &entity.Income{
        UserID:      uuid.New(),
        Description: "Salário",
        Amount:      decimal.NewFromFloat(7500),
        Type:        entity.IncomeTypeSalary,
        Competence:  "2026-06",
        Recurrent:   true,
    }

    err := repo.Create(context.Background(), income)

    assert.NoError(t, err)
    assert.NotEmpty(t, income.ID) // preenchido pelo BeforeCreate hook
}
```

### Convenções de integração
- Um `setupTestDB` por pacote de teste — não compartilhe containers entre pacotes
- Limpe os dados entre testes com `t.Cleanup` ou `db.Exec("TRUNCATE ...")`
- AutoMigrate apenas as entidades usadas no pacote de teste em questão
- Nunca use SQLite como substituto — os tipos PostgreSQL importam (ex: `decimal`, `uuid`)

---

## Nomeação de testes

```
Test<Entity/UseCase>_<Operation>        → TestIncomeUseCase_Create
Test<Entity/UseCase>_<Operation>_<case> → TestIncomeUseCase_Create_InvalidAmount
TestIncomeRepository_FindByCompetence
```

---

## Cobertura mínima esperada

| Camada | Meta |
|--------|------|
| Use cases com lógica de negócio | 80% |
| Repositórios | caminhos feliz + soft delete + filtros principais |
| Cálculos de saldo (SAL) | 100% dos cenários de regra |

Para verificar cobertura:
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

---

## Dependências de teste

```
github.com/testcontainers/testcontainers-go
github.com/testcontainers/testcontainers-go/modules/postgres
github.com/stretchr/testify
```

Verifique que estão em `go.mod` antes de escrever o primeiro teste de integração:
```bash
go get github.com/testcontainers/testcontainers-go
go get github.com/testcontainers/testcontainers-go/modules/postgres
```
