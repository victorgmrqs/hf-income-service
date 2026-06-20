package income

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestDeleteIncome_Success(t *testing.T) {
	repo := &mockIncomeRepository{
		DeleteFn: func(context.Context, uuid.UUID) error { return nil },
	}
	uc := NewDeleteUseCase(repo, testLogger())

	if err := uc.Execute(context.Background(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteIncome_NotFound(t *testing.T) {
	repo := &mockIncomeRepository{
		DeleteFn: func(context.Context, uuid.UUID) error { return gorm.ErrRecordNotFound },
	}
	uc := NewDeleteUseCase(repo, testLogger())

	err := uc.Execute(context.Background(), uuid.New())
	if !errors.Is(err, ErrIncomeNotFound) {
		t.Fatalf("err = %v, want ErrIncomeNotFound", err)
	}
}
