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

// newReductionGoalTestDB sobe um PostgreSQL via testcontainers e retorna uma conexão
// GORM já migrada com a tabela reduction_goals. Faz t.Skip se o Docker/testcontainers
// não estiver disponível (ambiente local sem Docker) — no CI o container sobe normalmente.
func newReductionGoalTestDB(t *testing.T) *gorm.DB {
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
	if err := db.AutoMigrate(&entity.ReductionGoal{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	// MET-04 (HF-68): índice único parcial — mesmo caminho de migração do main.go.
	if err := EnsureReductionGoalIndexes(db); err != nil {
		t.Fatalf("ensure indexes: %v", err)
	}
	return db
}

func validGoal(userID, categoryID uuid.UUID, competence string) *entity.ReductionGoal {
	return &entity.ReductionGoal{
		UserID:       userID,
		CategoryID:   categoryID,
		Competence:   competence,
		TargetAmount: decimal.RequireFromString("400.00"),
	}
}

func TestReductionGoalRepository_Create_Success(t *testing.T) {
	db := newReductionGoalTestDB(t)
	repo := NewReductionGoalRepository(db)
	ctx := context.Background()
	userID, categoryID := uuid.New(), uuid.New()

	goal := validGoal(userID, categoryID, "2026-06")
	if err := repo.Create(ctx, goal); err != nil {
		t.Fatalf("create: %v", err)
	}
	if goal.ID == uuid.Nil {
		t.Fatal("expected generated UUID")
	}

	got, err := repo.GetByID(ctx, goal.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if !got.TargetAmount.Equal(decimal.RequireFromString("400.00")) {
		t.Errorf("target_amount = %s, want 400.00", got.TargetAmount)
	}
	if got.Achieved != nil {
		t.Errorf("achieved = %v, want nil (mês em curso)", got.Achieved)
	}
}

// TestReductionGoalRepository_Create_DuplicateConflict cobre MET-04: a combinação
// (user_id, category_id, competence) deve ser única entre registros ativos.
func TestReductionGoalRepository_Create_DuplicateConflict(t *testing.T) {
	db := newReductionGoalTestDB(t)
	repo := NewReductionGoalRepository(db)
	ctx := context.Background()
	userID, categoryID := uuid.New(), uuid.New()

	original := validGoal(userID, categoryID, "2026-06")
	if err := repo.Create(ctx, original); err != nil {
		t.Fatalf("create: %v", err)
	}

	dup := validGoal(userID, categoryID, "2026-06")
	if err := repo.Create(ctx, dup); err == nil {
		t.Fatal("expected unique constraint violation on (user_id, category_id, competence)")
	}

	// Outra categoria na mesma competência deve funcionar normalmente.
	otherCategory := validGoal(userID, uuid.New(), "2026-06")
	if err := repo.Create(ctx, otherCategory); err != nil {
		t.Fatalf("create other category: %v", err)
	}
}

func TestReductionGoalRepository_ListByUserAndCompetence_ExcludesDeleted(t *testing.T) {
	db := newReductionGoalTestDB(t)
	repo := NewReductionGoalRepository(db)
	ctx := context.Background()
	userID := uuid.New()

	kept := validGoal(userID, uuid.New(), "2026-06")
	if err := repo.Create(ctx, kept); err != nil {
		t.Fatalf("create kept: %v", err)
	}
	deleted := validGoal(userID, uuid.New(), "2026-06")
	if err := repo.Create(ctx, deleted); err != nil {
		t.Fatalf("create deleted: %v", err)
	}
	// Outra competência — fora da listagem.
	other := validGoal(userID, uuid.New(), "2026-07")
	if err := repo.Create(ctx, other); err != nil {
		t.Fatalf("create other competence: %v", err)
	}

	if err := repo.Delete(ctx, deleted.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	list, err := repo.ListByUserAndCompetence(ctx, userID, "2026-06")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].ID != kept.ID {
		t.Fatalf("list = %+v, want apenas %s", list, kept.ID)
	}

	// MET-04: soft delete libera a constraint — recriar a mesma combinação funciona
	// (índice único é PARCIAL, WHERE deleted_at IS NULL).
	recreated := validGoal(userID, deleted.CategoryID, "2026-06")
	if err := repo.Create(ctx, recreated); err != nil {
		t.Fatalf("recreate after soft delete: %v", err)
	}
}

func TestReductionGoalRepository_SetAchieved_True(t *testing.T) {
	db := newReductionGoalTestDB(t)
	repo := NewReductionGoalRepository(db)
	ctx := context.Background()

	goal := validGoal(uuid.New(), uuid.New(), "2026-06")
	if err := repo.Create(ctx, goal); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := repo.SetAchieved(ctx, goal.ID, true); err != nil {
		t.Fatalf("set achieved: %v", err)
	}
	got, err := repo.GetByID(ctx, goal.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Achieved == nil || !*got.Achieved {
		t.Errorf("achieved = %v, want true", got.Achieved)
	}
}

func TestReductionGoalRepository_SetAchieved_False(t *testing.T) {
	db := newReductionGoalTestDB(t)
	repo := NewReductionGoalRepository(db)
	ctx := context.Background()

	goal := validGoal(uuid.New(), uuid.New(), "2026-06")
	if err := repo.Create(ctx, goal); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := repo.SetAchieved(ctx, goal.ID, false); err != nil {
		t.Fatalf("set achieved: %v", err)
	}
	got, err := repo.GetByID(ctx, goal.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Achieved == nil || *got.Achieved {
		t.Errorf("achieved = %v, want false", got.Achieved)
	}

	// SetAchieved em ID inexistente → ErrRecordNotFound.
	if err := repo.SetAchieved(ctx, uuid.New(), true); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("set achieved missing: err = %v, want ErrRecordNotFound", err)
	}
}

// TestReductionGoalRepository_ListWithNullAchieved cobre a base do job de
// fechamento do mês (MET-06): apenas metas com achieved IS NULL retornam.
func TestReductionGoalRepository_ListWithNullAchieved(t *testing.T) {
	db := newReductionGoalTestDB(t)
	repo := NewReductionGoalRepository(db)
	ctx := context.Background()

	pending1 := validGoal(uuid.New(), uuid.New(), "2026-06")
	pending2 := validGoal(uuid.New(), uuid.New(), "2026-06")
	closed := validGoal(uuid.New(), uuid.New(), "2026-06")
	deleted := validGoal(uuid.New(), uuid.New(), "2026-06")
	otherCompetence := validGoal(uuid.New(), uuid.New(), "2026-07")
	for _, g := range []*entity.ReductionGoal{pending1, pending2, closed, deleted, otherCompetence} {
		if err := repo.Create(ctx, g); err != nil {
			t.Fatalf("create seed: %v", err)
		}
	}
	if err := repo.SetAchieved(ctx, closed.ID, true); err != nil {
		t.Fatalf("set achieved: %v", err)
	}
	// Meta pendente (achieved: null) mas soft-deletada não deve entrar no job de
	// fechamento (MET-06) — o job não deve reviver nem processar metas excluídas.
	if err := repo.Delete(ctx, deleted.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	pending, err := repo.ListWithNullAchievedByCompetence(ctx, "2026-06")
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) != 2 {
		t.Fatalf("pending = %d, want 2 (pending1 e pending2)", len(pending))
	}
	ids := map[uuid.UUID]bool{pending[0].ID: true, pending[1].ID: true}
	if !ids[pending1.ID] || !ids[pending2.ID] {
		t.Errorf("pending ids = %v, want %s e %s", ids, pending1.ID, pending2.ID)
	}
	if ids[deleted.ID] {
		t.Error("meta soft-deletada apareceu em ListWithNullAchievedByCompetence")
	}

	// Sem metas pendentes na competência → lista vazia, sem erro.
	empty, err := repo.ListWithNullAchievedByCompetence(ctx, "2099-01")
	if err != nil {
		t.Fatalf("list pending (empty): %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("empty = %d, want 0", len(empty))
	}
}

func TestReductionGoalRepository_ExistsByUserCategoryAndCompetence(t *testing.T) {
	db := newReductionGoalTestDB(t)
	repo := NewReductionGoalRepository(db)
	ctx := context.Background()
	userID, categoryID := uuid.New(), uuid.New()

	exists, err := repo.ExistsByUserCategoryAndCompetence(ctx, userID, categoryID, "2026-06")
	if err != nil {
		t.Fatalf("exists (empty): %v", err)
	}
	if exists {
		t.Error("exists = true antes de criar, want false")
	}

	goal := validGoal(userID, categoryID, "2026-06")
	if err := repo.Create(ctx, goal); err != nil {
		t.Fatalf("create: %v", err)
	}

	exists, err = repo.ExistsByUserCategoryAndCompetence(ctx, userID, categoryID, "2026-06")
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if !exists {
		t.Error("exists = false após criar, want true")
	}

	// Após soft delete, deixa de existir (deletados são ignorados) — MET-04.
	if err := repo.Delete(ctx, goal.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	exists, err = repo.ExistsByUserCategoryAndCompetence(ctx, userID, categoryID, "2026-06")
	if err != nil {
		t.Fatalf("exists (deleted): %v", err)
	}
	if exists {
		t.Error("exists = true após soft delete, want false")
	}
}

func TestReductionGoalRepository_UpdatePreviousAmount(t *testing.T) {
	db := newReductionGoalTestDB(t)
	repo := NewReductionGoalRepository(db)
	ctx := context.Background()

	goal := validGoal(uuid.New(), uuid.New(), "2026-06")
	if err := repo.Create(ctx, goal); err != nil {
		t.Fatalf("create: %v", err)
	}
	if goal.PreviousAmount != nil {
		t.Fatalf("previous_amount inicial = %v, want nil", goal.PreviousAmount)
	}

	if err := repo.UpdatePreviousAmount(ctx, goal.ID, decimal.RequireFromString("520.00")); err != nil {
		t.Fatalf("update previous amount: %v", err)
	}
	got, err := repo.GetByID(ctx, goal.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.PreviousAmount == nil || !got.PreviousAmount.Equal(decimal.RequireFromString("520.00")) {
		t.Errorf("previous_amount = %v, want 520.00", got.PreviousAmount)
	}

	// ID inexistente → ErrRecordNotFound.
	if err := repo.UpdatePreviousAmount(ctx, uuid.New(), decimal.RequireFromString("1.00")); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("update previous amount missing: err = %v, want ErrRecordNotFound", err)
	}
}

func TestReductionGoalRepository_Update(t *testing.T) {
	db := newReductionGoalTestDB(t)
	repo := NewReductionGoalRepository(db)
	ctx := context.Background()

	goal := validGoal(uuid.New(), uuid.New(), "2026-06")
	if err := repo.Create(ctx, goal); err != nil {
		t.Fatalf("create: %v", err)
	}

	goal.TargetAmount = decimal.RequireFromString("350.00")
	if err := repo.Update(ctx, goal); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err := repo.GetByID(ctx, goal.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !got.TargetAmount.Equal(decimal.RequireFromString("350.00")) {
		t.Errorf("target_amount após update = %s, want 350.00", got.TargetAmount)
	}
}

func TestReductionGoalRepository_GetByID_NotFound(t *testing.T) {
	db := newReductionGoalTestDB(t)
	repo := NewReductionGoalRepository(db)
	ctx := context.Background()

	if _, err := repo.GetByID(ctx, uuid.New()); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("get missing: err = %v, want ErrRecordNotFound", err)
	}
}

func TestReductionGoalRepository_Delete_NotFound(t *testing.T) {
	db := newReductionGoalTestDB(t)
	repo := NewReductionGoalRepository(db)
	ctx := context.Background()

	if err := repo.Delete(ctx, uuid.New()); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("delete missing: err = %v, want ErrRecordNotFound", err)
	}
}
