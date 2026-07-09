package entity_test

import (
	"context"
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
	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
)

func newEntityTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("hf_entity_test"),
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
	if err := db.AutoMigrate(
		&entity.Income{},
		&entity.GlobalBudget{},
		&entity.ReductionGoal{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	// ORC-01 (HF-44): a unicidade de GlobalBudget é um índice único PARCIAL
	// criado fora do AutoMigrate — mesmo caminho de migração do main.go.
	if err := repository.EnsureGlobalBudgetIndexes(db); err != nil {
		t.Fatalf("ensure global budget indexes: %v", err)
	}
	return db
}

func TestAutoMigrate_CreatesAllTables(t *testing.T) {
	db := newEntityTestDB(t)

	tables := []string{"incomes", "global_budgets", "reduction_goals"}
	for _, table := range tables {
		if !db.Migrator().HasTable(table) {
			t.Errorf("table %q not created by AutoMigrate", table)
		}
	}
}

func TestGlobalBudget_UniqueConstraint_UserAndCompetence(t *testing.T) {
	db := newEntityTestDB(t)
	userID := uuid.New()

	first := &entity.GlobalBudget{
		UserID:       userID,
		Competence:   "2026-06",
		Ceiling:      decimal.NewFromFloat(3000),
		AutoAdjusted: false,
	}
	if err := db.Create(first).Error; err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	duplicate := &entity.GlobalBudget{
		UserID:       userID,
		Competence:   "2026-06",
		Ceiling:      decimal.NewFromFloat(4000),
		AutoAdjusted: false,
	}
	err := db.Create(duplicate).Error
	if err == nil {
		t.Error("expected unique constraint violation, got nil error")
	}
}

func TestReductionGoal_UniqueConstraint_UserCategoryCompetence(t *testing.T) {
	db := newEntityTestDB(t)
	userID := uuid.New()
	categoryID := uuid.New()

	first := &entity.ReductionGoal{
		UserID:       userID,
		CategoryID:   categoryID,
		Competence:   "2026-06",
		TargetAmount: decimal.NewFromFloat(500),
	}
	if err := db.Create(first).Error; err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	duplicate := &entity.ReductionGoal{
		UserID:       userID,
		CategoryID:   categoryID,
		Competence:   "2026-06",
		TargetAmount: decimal.NewFromFloat(600),
	}
	err := db.Create(duplicate).Error
	if err == nil {
		t.Error("expected unique constraint violation, got nil error")
	}
}
