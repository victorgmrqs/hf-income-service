package income

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
)

func recurrentOriginal() entity.Income {
	return entity.Income{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		Description: "Salário",
		Amount:      decimal.RequireFromString("5000.00"),
		Date:        time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC),
		Competence:  "2026-06",
		Type:        entity.IncomeTypeSalary,
		Recurrent:   true,
	}
}

func TestPropagate_CreatesCopies(t *testing.T) {
	original := recurrentOriginal()
	var created *entity.Income
	repo := &mockIncomeRepository{
		ListRecurrentFn: func(_ context.Context, competence string) ([]entity.Income, error) {
			if competence != "2026-06" {
				t.Errorf("previous competence = %s, want 2026-06", competence)
			}
			return []entity.Income{original}, nil
		},
		ExistsFn: func(context.Context, uuid.UUID, string) (bool, error) { return false, nil },
		CreateFn: func(_ context.Context, inc *entity.Income) error {
			inc.ID = uuid.New()
			created = inc
			return nil
		},
	}
	uc := NewPropagateUseCase(repo, testLogger())

	out, err := uc.Execute(context.Background(), PropagateInput{Competence: "2026-07"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Propagated != 1 {
		t.Fatalf("propagated = %d, want 1", out.Propagated)
	}
	if created.Competence != "2026-07" {
		t.Errorf("competence = %s, want 2026-07", created.Competence)
	}
	if created.OriginID == nil || *created.OriginID != original.ID {
		t.Errorf("origin_id = %v, want %s", created.OriginID, original.ID)
	}
	if !created.Recurrent {
		t.Error("copy should remain recurrent")
	}
	if created.Date.Month() != time.July || created.Date.Day() != 5 {
		t.Errorf("date = %s, want July day 5", created.Date.Format("2006-01-02"))
	}
}

func TestPropagate_Idempotent(t *testing.T) {
	original := recurrentOriginal()
	repo := &mockIncomeRepository{
		ListRecurrentFn: func(context.Context, string) ([]entity.Income, error) {
			return []entity.Income{original}, nil
		},
		ExistsFn: func(context.Context, uuid.UUID, string) (bool, error) { return true, nil },
		CreateFn: func(context.Context, *entity.Income) error {
			t.Fatal("Create should not be called when copy already exists")
			return nil
		},
	}
	uc := NewPropagateUseCase(repo, testLogger())

	out, err := uc.Execute(context.Background(), PropagateInput{Competence: "2026-07"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Propagated != 0 {
		t.Errorf("propagated = %d, want 0", out.Propagated)
	}
}

func TestPropagate_NoRecurrent(t *testing.T) {
	repo := &mockIncomeRepository{
		ListRecurrentFn: func(context.Context, string) ([]entity.Income, error) {
			return []entity.Income{}, nil
		},
	}
	uc := NewPropagateUseCase(repo, testLogger())

	out, err := uc.Execute(context.Background(), PropagateInput{Competence: "2026-07"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Propagated != 0 {
		t.Errorf("propagated = %d, want 0", out.Propagated)
	}
}

func TestPropagate_InvalidCompetence(t *testing.T) {
	repo := &mockIncomeRepository{
		ListRecurrentFn: func(context.Context, string) ([]entity.Income, error) {
			t.Fatal("ListRecurrent should not be called on invalid competence")
			return nil, nil
		},
	}
	uc := NewPropagateUseCase(repo, testLogger())

	_, err := uc.Execute(context.Background(), PropagateInput{Competence: "2026/13"})
	if !errors.Is(err, ErrInvalidCompetence) {
		t.Fatalf("err = %v, want ErrInvalidCompetence", err)
	}
}
