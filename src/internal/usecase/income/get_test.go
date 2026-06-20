package income

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
)

func TestGetIncome_Success(t *testing.T) {
	id := uuid.New()
	repo := &mockIncomeRepository{
		FindByIDFn: func(_ context.Context, gotID uuid.UUID) (*entity.Income, error) {
			return &entity.Income{ID: gotID, Type: entity.IncomeTypeSalary, Competence: "2026-06"}, nil
		},
	}
	uc := NewGetUseCase(repo, testLogger())

	out, err := uc.Execute(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ID != id {
		t.Errorf("id = %s, want %s", out.ID, id)
	}
}

func TestGetIncome_NotFound(t *testing.T) {
	repo := &mockIncomeRepository{
		FindByIDFn: func(context.Context, uuid.UUID) (*entity.Income, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}
	uc := NewGetUseCase(repo, testLogger())

	_, err := uc.Execute(context.Background(), uuid.New())
	if !errors.Is(err, ErrIncomeNotFound) {
		t.Fatalf("err = %v, want ErrIncomeNotFound", err)
	}
}
