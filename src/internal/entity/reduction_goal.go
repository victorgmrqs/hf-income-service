package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ReductionGoal representa uma meta de redução de gastos por categoria (MET).
// MET-04: combinação (user_id, category_id, competence) é única.
// previous_amount é nullable — preenchido com null quando hf-transaction-service
// está indisponível na criação (degradação sem erro HTTP).
// achieved é nullable: null = mês em andamento; true/false = fechamento aplicado.
type ReductionGoal struct {
	ID             uuid.UUID        `gorm:"type:uuid;primaryKey" json:"id"`
	UserID         uuid.UUID        `gorm:"type:uuid;not null;index:idx_reduction_goals_user_competence,priority:1;uniqueIndex:idx_reduction_goals_user_category_competence,priority:1" json:"user_id"`
	CategoryID     uuid.UUID        `gorm:"type:uuid;not null;uniqueIndex:idx_reduction_goals_user_category_competence,priority:2" json:"category_id"`
	Competence     string           `gorm:"size:7;not null;index:idx_reduction_goals_user_competence,priority:2;uniqueIndex:idx_reduction_goals_user_category_competence,priority:3" json:"competence"`
	TargetAmount   decimal.Decimal  `gorm:"type:decimal(10,2);not null" json:"target_amount"`
	PreviousAmount *decimal.Decimal `gorm:"type:decimal(10,2)" json:"previous_amount"`
	Achieved       *bool            `gorm:"default:null" json:"achieved"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
	DeletedAt      gorm.DeletedAt   `gorm:"index" json:"deleted_at"`
}

func (g *ReductionGoal) BeforeCreate(tx *gorm.DB) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return nil
}
