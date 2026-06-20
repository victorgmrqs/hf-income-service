package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
)

// newTestDB sobe um PostgreSQL via testcontainers e retorna uma conexão GORM já
// migrada. Faz t.Skip se o Docker/testcontainers não estiver disponível (ambiente
// local sem Docker) — no CI o container sobe normalmente.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("hf_income_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Skipf("docker/testcontainers indisponível: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm open: %v", err)
	}
	if err := db.AutoMigrate(&entity.Income{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return db
}

func TestIncomeRepository_Integration(t *testing.T) {
	db := newTestDB(t)
	repo := NewIncomeRepository(db)
	ctx := context.Background()
	userID := uuid.New()

	// Create
	inc := &entity.Income{
		UserID:      userID,
		Description: "Salário",
		Amount:      decimal.RequireFromString("5000.00"),
		Date:        time.Now(),
		Competence:  "2026-06",
		Type:        entity.IncomeTypeSalary,
	}
	if err := repo.Create(ctx, inc); err != nil {
		t.Fatalf("create: %v", err)
	}
	if inc.ID == uuid.Nil {
		t.Fatal("expected generated UUID")
	}

	// FindByID
	got, err := repo.FindByID(ctx, inc.ID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if !got.Amount.Equal(decimal.RequireFromString("5000.00")) {
		t.Errorf("amount = %s, want 5000.00", got.Amount)
	}

	// List por competência
	inc2 := &entity.Income{
		UserID:      userID,
		Description: "Freela",
		Amount:      decimal.RequireFromString("2500.00"),
		Date:        time.Now(),
		Competence:  "2026-06",
		Type:        entity.IncomeTypeFreelance,
	}
	if err := repo.Create(ctx, inc2); err != nil {
		t.Fatalf("create 2: %v", err)
	}
	list, err := repo.ListByUserAndCompetence(ctx, userID, "2026-06")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("list len = %d, want 2", len(list))
	}

	// Update
	got.Description = "Salário corrigido"
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("update: %v", err)
	}

	// Soft delete: registro some das buscas/listagens
	if err := repo.Delete(ctx, inc.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.FindByID(ctx, inc.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("after delete: err = %v, want ErrRecordNotFound", err)
	}
	list2, err := repo.ListByUserAndCompetence(ctx, userID, "2026-06")
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(list2) != 1 {
		t.Errorf("list after delete = %d, want 1", len(list2))
	}

	// Delete inexistente → ErrRecordNotFound
	if err := repo.Delete(ctx, uuid.New()); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("delete missing: err = %v, want ErrRecordNotFound", err)
	}
}
