package entity_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
)

func TestIncome_BeforeCreate_GeneratesUUID(t *testing.T) {
	income := &entity.Income{}
	if err := income.BeforeCreate(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if income.ID == uuid.Nil {
		t.Error("expected UUID to be generated, got nil")
	}
}

func TestIncome_BeforeCreate_PreservesExistingUUID(t *testing.T) {
	existing := uuid.New()
	income := &entity.Income{ID: existing}
	if err := income.BeforeCreate(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if income.ID != existing {
		t.Errorf("got %v, want %v", income.ID, existing)
	}
}

func TestGlobalBudget_BeforeCreate_GeneratesUUID(t *testing.T) {
	budget := &entity.GlobalBudget{}
	if err := budget.BeforeCreate(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if budget.ID == uuid.Nil {
		t.Error("expected UUID to be generated, got nil")
	}
}

func TestReductionGoal_BeforeCreate_GeneratesUUID(t *testing.T) {
	goal := &entity.ReductionGoal{}
	if err := goal.BeforeCreate(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if goal.ID == uuid.Nil {
		t.Error("expected UUID to be generated, got nil")
	}
}
