package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// GlobalBudget representa o teto mensal de gastos de um usuário (ORC).
// Não usa soft delete — substituição direta ao editar (ENTITIES.md).
// ORC-01: combinação (user_id, competence) é única.
type GlobalBudget struct {
	ID           uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	UserID       uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex:idx_global_budgets_user_competence,priority:1" json:"user_id"`
	Competence   string          `gorm:"size:7;not null;uniqueIndex:idx_global_budgets_user_competence,priority:2" json:"competence"`
	Ceiling      decimal.Decimal `gorm:"type:decimal(10,2);not null" json:"ceiling"`
	AutoAdjusted bool            `gorm:"not null;default:false" json:"auto_adjusted"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

func (b *GlobalBudget) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}
