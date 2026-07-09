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

// newGlobalBudgetTestDB sobe um PostgreSQL via testcontainers e retorna uma conexão
// GORM já migrada com a tabela global_budgets. Faz t.Skip se o Docker/testcontainers
// não estiver disponível (ambiente local sem Docker) — no CI o container sobe normalmente.
func newGlobalBudgetTestDB(t *testing.T) *gorm.DB {
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
	if err := db.AutoMigrate(&entity.GlobalBudget{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	// ORC-01 (HF-44): índice único parcial — mesmo caminho de migração do main.go.
	if err := EnsureGlobalBudgetIndexes(db); err != nil {
		t.Fatalf("ensure indexes: %v", err)
	}
	return db
}

func TestGlobalBudgetRepository_Integration(t *testing.T) {
	db := newGlobalBudgetTestDB(t)
	repo := NewGlobalBudgetRepository(db)
	ctx := context.Background()
	userID := uuid.New()

	// Create
	budget := &entity.GlobalBudget{
		UserID:     userID,
		Competence: "2026-06",
		Ceiling:    decimal.RequireFromString("5000.00"),
	}
	if err := repo.Create(ctx, budget); err != nil {
		t.Fatalf("create: %v", err)
	}
	if budget.ID == uuid.Nil {
		t.Fatal("expected generated UUID")
	}

	// GetByID
	byID, err := repo.GetByID(ctx, budget.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if !byID.Ceiling.Equal(decimal.RequireFromString("5000.00")) {
		t.Errorf("ceiling = %s, want 5000.00", byID.Ceiling)
	}

	// GetByUserAndCompetence
	byComp, err := repo.GetByUserAndCompetence(ctx, userID, "2026-06")
	if err != nil {
		t.Fatalf("get by user/competence: %v", err)
	}
	if byComp.ID != budget.ID {
		t.Errorf("id = %s, want %s", byComp.ID, budget.ID)
	}

	// Update — força auto_adjusted = false (ORC-05) e novo ceiling
	byComp.Ceiling = decimal.RequireFromString("4800.00")
	byComp.AutoAdjusted = false
	if err := repo.Update(ctx, byComp); err != nil {
		t.Fatalf("update: %v", err)
	}
	updated, _ := repo.GetByID(ctx, budget.ID)
	if !updated.Ceiling.Equal(decimal.RequireFromString("4800.00")) {
		t.Errorf("ceiling após update = %s, want 4800.00", updated.Ceiling)
	}

	// ORC-01: unique (user_id, competence) — segundo teto na mesma competência falha
	dup := &entity.GlobalBudget{
		UserID:     userID,
		Competence: "2026-06",
		Ceiling:    decimal.RequireFromString("100.00"),
	}
	if err := repo.Create(ctx, dup); err == nil {
		t.Fatal("expected unique constraint violation on (user_id, competence)")
	}

	// Mesma competência para outro usuário deve funcionar
	otherUser := &entity.GlobalBudget{
		UserID:     uuid.New(),
		Competence: "2026-06",
		Ceiling:    decimal.RequireFromString("3000.00"),
	}
	if err := repo.Create(ctx, otherUser); err != nil {
		t.Fatalf("create other user same competence: %v", err)
	}

	// GetByID inexistente → ErrRecordNotFound
	if _, err := repo.GetByID(ctx, uuid.New()); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("get missing: err = %v, want ErrRecordNotFound", err)
	}
	// GetByUserAndCompetence sem registro → ErrRecordNotFound
	if _, err := repo.GetByUserAndCompetence(ctx, uuid.New(), "2099-01"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("get missing by competence: err = %v, want ErrRecordNotFound", err)
	}
}

func TestGlobalBudgetRepository_Upsert_CreatesWhenNotExists(t *testing.T) {
	db := newGlobalBudgetTestDB(t)
	repo := NewGlobalBudgetRepository(db)
	ctx := context.Background()

	budget := &entity.GlobalBudget{
		UserID:       uuid.New(),
		Competence:   "2026-07",
		Ceiling:      decimal.RequireFromString("4200.00"),
		AutoAdjusted: true,
	}
	if err := repo.Upsert(ctx, budget); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if budget.ID == uuid.Nil {
		t.Fatal("expected generated UUID on upsert-create")
	}
	got, err := repo.GetByUserAndCompetence(ctx, budget.UserID, "2026-07")
	if err != nil {
		t.Fatalf("get after upsert: %v", err)
	}
	if !got.AutoAdjusted {
		t.Error("auto_adjusted = false, want true")
	}
}

func TestGlobalBudgetRepository_Upsert_UpdatesWhenExists(t *testing.T) {
	db := newGlobalBudgetTestDB(t)
	repo := NewGlobalBudgetRepository(db)
	ctx := context.Background()
	userID := uuid.New()

	original := &entity.GlobalBudget{
		UserID:     userID,
		Competence: "2026-07",
		Ceiling:    decimal.RequireFromString("5000.00"),
	}
	if err := repo.Create(ctx, original); err != nil {
		t.Fatalf("create: %v", err)
	}

	adjusted := &entity.GlobalBudget{
		UserID:       userID,
		Competence:   "2026-07",
		Ceiling:      decimal.RequireFromString("4200.00"),
		AutoAdjusted: true,
	}
	if err := repo.Upsert(ctx, adjusted); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	// Atualiza o registro existente — mesmo ID, sem duplicar (idempotência ORC-03/05).
	if adjusted.ID != original.ID {
		t.Errorf("upsert created new record: id %s != %s", adjusted.ID, original.ID)
	}
	got, err := repo.GetByUserAndCompetence(ctx, userID, "2026-07")
	if err != nil {
		t.Fatalf("get after upsert: %v", err)
	}
	if !got.Ceiling.Equal(decimal.RequireFromString("4200.00")) {
		t.Errorf("ceiling = %s, want 4200.00", got.Ceiling)
	}
	if !got.AutoAdjusted {
		t.Error("auto_adjusted = false, want true")
	}
}

func TestGlobalBudgetRepository_ExistsByUserAndCompetence(t *testing.T) {
	db := newGlobalBudgetTestDB(t)
	repo := NewGlobalBudgetRepository(db)
	ctx := context.Background()
	userID := uuid.New()

	exists, err := repo.ExistsByUserAndCompetence(ctx, userID, "2026-06")
	if err != nil {
		t.Fatalf("exists (empty): %v", err)
	}
	if exists {
		t.Error("exists = true antes de criar, want false")
	}

	budget := &entity.GlobalBudget{
		UserID:     userID,
		Competence: "2026-06",
		Ceiling:    decimal.RequireFromString("5000.00"),
	}
	if err := repo.Create(ctx, budget); err != nil {
		t.Fatalf("create: %v", err)
	}

	exists, err = repo.ExistsByUserAndCompetence(ctx, userID, "2026-06")
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if !exists {
		t.Error("exists = false após criar, want true")
	}

	// Após soft delete, deixa de existir (deletados são ignorados).
	if err := repo.Delete(ctx, budget.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	exists, err = repo.ExistsByUserAndCompetence(ctx, userID, "2026-06")
	if err != nil {
		t.Fatalf("exists (deleted): %v", err)
	}
	if exists {
		t.Error("exists = true após soft delete, want false")
	}
}

func TestGlobalBudgetRepository_SoftDelete_HiddenAndRecreatable(t *testing.T) {
	db := newGlobalBudgetTestDB(t)
	repo := NewGlobalBudgetRepository(db)
	ctx := context.Background()
	userID := uuid.New()

	budget := &entity.GlobalBudget{
		UserID:     userID,
		Competence: "2026-06",
		Ceiling:    decimal.RequireFromString("5000.00"),
	}
	if err := repo.Create(ctx, budget); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Soft delete: some das consultas.
	if err := repo.Delete(ctx, budget.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.GetByID(ctx, budget.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("get by id after delete: err = %v, want ErrRecordNotFound", err)
	}
	if _, err := repo.GetByUserAndCompetence(ctx, userID, "2026-06"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("get by competence after delete: err = %v, want ErrRecordNotFound", err)
	}

	// ORC-01 (HF-44): o índice único PARCIAL permite recriar o teto da mesma
	// competência após o soft delete — um índice full bloquearia este Create.
	recreated := &entity.GlobalBudget{
		UserID:     userID,
		Competence: "2026-06",
		Ceiling:    decimal.RequireFromString("4500.00"),
	}
	if err := repo.Create(ctx, recreated); err != nil {
		t.Fatalf("recreate after soft delete: %v", err)
	}

	// Delete de registro já deletado/inexistente → ErrRecordNotFound.
	if err := repo.Delete(ctx, budget.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("delete deleted: err = %v, want ErrRecordNotFound", err)
	}
}
