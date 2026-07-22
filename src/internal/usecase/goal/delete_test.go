package goal

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestGoalDelete_Success(t *testing.T) {
	repo := &mockReductionGoalRepository{
		DeleteFn: func(context.Context, uuid.UUID) error { return nil },
	}
	uc := NewDeleteUseCase(repo, testLogger())

	if err := uc.Execute(context.Background(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGoalDelete_NotFound(t *testing.T) {
	repo := &mockReductionGoalRepository{
		DeleteFn: func(context.Context, uuid.UUID) error { return gorm.ErrRecordNotFound },
	}
	uc := NewDeleteUseCase(repo, testLogger())

	err := uc.Execute(context.Background(), uuid.New())
	if !errors.Is(err, ErrGoalNotFound) {
		t.Fatalf("err = %v, want ErrGoalNotFound", err)
	}
}
