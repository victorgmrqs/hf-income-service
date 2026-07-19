package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
	"github.com/victorgmrqs/hf-income-service/src/internal/repository"
	budgetUseCase "github.com/victorgmrqs/hf-income-service/src/internal/usecase/global_budget"
	incomeUseCase "github.com/victorgmrqs/hf-income-service/src/internal/usecase/income"
	"github.com/victorgmrqs/hf-income-service/src/pkg/observability"
)

func testLogger() *slog.Logger { return observability.NewLogger("test") }

// fakePropagate implementa income.PropagateUseCase com contagem thread-safe.
type fakePropagate struct {
	mu    sync.Mutex
	calls []string
	err   error
}

func (f *fakePropagate) Execute(_ context.Context, input incomeUseCase.PropagateInput) (*incomeUseCase.PropagateOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, input.Competence)
	if f.err != nil {
		return nil, f.err
	}
	return &incomeUseCase.PropagateOutput{Propagated: 2}, nil
}

func (f *fakePropagate) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// fakeAutoAdjust implementa global_budget.AutoAdjustUseCase; errFor injeta falha
// por usuário específico.
type fakeAutoAdjust struct {
	mu     sync.Mutex
	calls  []budgetUseCase.AutoAdjustInput
	errFor map[uuid.UUID]error
}

func (f *fakeAutoAdjust) Execute(_ context.Context, input budgetUseCase.AutoAdjustInput) (*budgetUseCase.GlobalBudgetOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, input)
	if err, ok := f.errFor[input.UserID]; ok {
		return nil, err
	}
	return &budgetUseCase.GlobalBudgetOutput{
		ID:         uuid.New(),
		UserID:     input.UserID,
		Competence: input.Competence,
		Ceiling:    decimal.RequireFromString("1000.00"),
	}, nil
}

func (f *fakeAutoAdjust) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// stubBudgetRepo implementa repository.GlobalBudgetRepository; o scheduler usa
// apenas ListUserIDsByCompetence.
type stubBudgetRepo struct {
	ListUserIDsFn func(ctx context.Context, competence string) ([]uuid.UUID, error)
}

var _ repository.GlobalBudgetRepository = (*stubBudgetRepo)(nil)

func (s *stubBudgetRepo) ListUserIDsByCompetence(ctx context.Context, competence string) ([]uuid.UUID, error) {
	return s.ListUserIDsFn(ctx, competence)
}
func (s *stubBudgetRepo) Create(context.Context, *entity.GlobalBudget) error {
	panic("unexpected Create")
}
func (s *stubBudgetRepo) GetByID(context.Context, uuid.UUID) (*entity.GlobalBudget, error) {
	panic("unexpected GetByID")
}
func (s *stubBudgetRepo) GetByUserAndCompetence(context.Context, uuid.UUID, string) (*entity.GlobalBudget, error) {
	panic("unexpected GetByUserAndCompetence")
}
func (s *stubBudgetRepo) Update(context.Context, *entity.GlobalBudget) error {
	panic("unexpected Update")
}
func (s *stubBudgetRepo) ExistsByUserAndCompetence(context.Context, uuid.UUID, string) (bool, error) {
	panic("unexpected Exists")
}
func (s *stubBudgetRepo) Upsert(context.Context, *entity.GlobalBudget) error {
	panic("unexpected Upsert")
}
func (s *stubBudgetRepo) Delete(context.Context, uuid.UUID) error { panic("unexpected Delete") }

// newTestScheduler monta o scheduler com relógio fixo e mocks; userA/userB são
// os elegíveis do auto-ajuste na competência anterior.
func newTestScheduler(now time.Time, users []uuid.UUID) (*Scheduler, *fakePropagate, *fakeAutoAdjust) {
	propagate := &fakePropagate{}
	autoAdjust := &fakeAutoAdjust{errFor: map[uuid.UUID]error{}}
	repo := &stubBudgetRepo{
		ListUserIDsFn: func(_ context.Context, _ string) ([]uuid.UUID, error) {
			return users, nil
		},
	}
	s := New(propagate, autoAdjust, repo, testLogger(), true, "UTC")
	s.nowFn = func() time.Time { return now }
	s.interval = 10 * time.Millisecond
	return s, propagate, autoAdjust
}

func TestScheduler_RunsJobsOnFirstOfMonth(t *testing.T) {
	userA, userB := uuid.New(), uuid.New()
	day1 := time.Date(2026, 8, 1, 6, 0, 0, 0, time.UTC)
	var askedCompetence string
	s, propagate, autoAdjust := newTestScheduler(day1, []uuid.UUID{userA, userB})
	base := s.budgetRepo.(*stubBudgetRepo).ListUserIDsFn
	s.budgetRepo.(*stubBudgetRepo).ListUserIDsFn = func(ctx context.Context, competence string) ([]uuid.UUID, error) {
		askedCompetence = competence
		return base(ctx, competence)
	}

	s.tick(context.Background())

	if propagate.count() != 1 || propagate.calls[0] != "2026-08" {
		t.Fatalf("propagate calls = %v, want 1x com 2026-08 (REC-04)", propagate.calls)
	}
	if askedCompetence != "2026-07" {
		t.Errorf("elegíveis buscados na competência %q, want 2026-07 (mês anterior)", askedCompetence)
	}
	if autoAdjust.count() != 2 {
		t.Fatalf("auto_adjust calls = %d, want 2 (um por usuário)", autoAdjust.count())
	}
	for _, call := range autoAdjust.calls {
		if call.Competence != "2026-08" {
			t.Errorf("auto_adjust competence = %q, want 2026-08 (destino)", call.Competence)
		}
	}
}

func TestScheduler_DoesNotRunOnOtherDays(t *testing.T) {
	day15 := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	s, propagate, autoAdjust := newTestScheduler(day15, []uuid.UUID{uuid.New()})

	s.tick(context.Background())

	if propagate.count() != 0 || autoAdjust.count() != 0 {
		t.Fatalf("jobs dispararam fora do dia 1: propagate=%d auto_adjust=%d", propagate.count(), autoAdjust.count())
	}
}

func TestScheduler_DoesNotRepeatSameCompetence(t *testing.T) {
	day1 := time.Date(2026, 8, 1, 6, 0, 0, 0, time.UTC)
	s, propagate, _ := newTestScheduler(day1, nil)

	s.tick(context.Background())
	s.tick(context.Background()) // segundo tick no mesmo dia 1

	if propagate.count() != 1 {
		t.Fatalf("propagate calls = %d, want 1 (dedupe por competência)", propagate.count())
	}
}

func TestScheduler_DisabledByEnv(t *testing.T) {
	day1 := time.Date(2026, 8, 1, 6, 0, 0, 0, time.UTC)
	s, propagate, _ := newTestScheduler(day1, nil)
	s.enabled = false

	done := make(chan struct{})
	go func() {
		s.Run(context.Background()) // deve retornar imediatamente
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run não retornou com scheduler desabilitado")
	}
	if propagate.count() != 0 {
		t.Fatalf("propagate disparou com SCHEDULER_ENABLED=false")
	}
}

func TestScheduler_JobFailureDoesNotStopOthers(t *testing.T) {
	userA, userB := uuid.New(), uuid.New()
	day1 := time.Date(2026, 8, 1, 6, 0, 0, 0, time.UTC)
	s, propagate, autoAdjust := newTestScheduler(day1, []uuid.UUID{userA, userB})
	propagate.err = errors.New("propagate exploded")
	autoAdjust.errFor[userA] = errors.New("upstream unavailable")

	s.tick(context.Background())

	// Falha do propagate não impede o auto-ajuste; falha do userA não impede o userB.
	if autoAdjust.count() != 2 {
		t.Fatalf("auto_adjust calls = %d, want 2 mesmo com falhas", autoAdjust.count())
	}
	var okForB bool
	for _, call := range autoAdjust.calls {
		if call.UserID == userB {
			okForB = true
		}
	}
	if !okForB {
		t.Error("userB não foi ajustado após falha do userA")
	}
}

func TestScheduler_StopsOnContextCancel(t *testing.T) {
	day15 := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	s, _, _ := newTestScheduler(day15, nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Run(ctx)
		close(done)
	}()

	time.Sleep(30 * time.Millisecond) // alguns ticks ociosos
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run não encerrou após o cancelamento do contexto")
	}
}
