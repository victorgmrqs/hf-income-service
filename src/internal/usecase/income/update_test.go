package income

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
)

func TestUpdateIncome_Success(t *testing.T) {
	id := uuid.New()
	repo := &mockIncomeRepository{
		FindByIDFn: func(_ context.Context, gotID uuid.UUID) (*entity.Income, error) {
			return &entity.Income{ID: gotID, Description: "antigo", Competence: "2026-06", Type: entity.IncomeTypeSalary}, nil
		},
		UpdateFn: func(context.Context, *entity.Income) error { return nil },
	}
	uc := NewUpdateUseCase(repo, testLogger())

	newDesc := "Salário corrigido"
	out, err := uc.Execute(context.Background(), UpdateInput{ID: id, Description: &newDesc})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Description != newDesc {
		t.Errorf("description = %s, want %s", out.Description, newDesc)
	}
}

func TestUpdateIncome_NotFound(t *testing.T) {
	repo := &mockIncomeRepository{
		FindByIDFn: func(context.Context, uuid.UUID) (*entity.Income, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}
	uc := NewUpdateUseCase(repo, testLogger())

	_, err := uc.Execute(context.Background(), UpdateInput{ID: uuid.New()})
	if !errors.Is(err, ErrIncomeNotFound) {
		t.Fatalf("err = %v, want ErrIncomeNotFound", err)
	}
}

func TestUpdateIncome_CannotEditPropagated(t *testing.T) {
	origin := uuid.New()
	repo := &mockIncomeRepository{
		FindByIDFn: func(_ context.Context, gotID uuid.UUID) (*entity.Income, error) {
			return &entity.Income{ID: gotID, OriginID: &origin, Competence: "2026-07"}, nil
		},
		UpdateFn: func(context.Context, *entity.Income) error {
			t.Fatal("repo.Update should not be called for propagated income")
			return nil
		},
	}
	uc := NewUpdateUseCase(repo, testLogger())

	newDesc := "x"
	_, err := uc.Execute(context.Background(), UpdateInput{ID: uuid.New(), Description: &newDesc})
	if !errors.Is(err, ErrCannotEditPropagatedIncome) {
		t.Fatalf("err = %v, want ErrCannotEditPropagatedIncome", err)
	}
}
