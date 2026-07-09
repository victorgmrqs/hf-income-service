package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// GlobalBudget representa o teto mensal de gastos de um usuário (ORC).
// Usa soft delete (decisão HF-44 — supersede o "sem soft delete" do HF-59).
// ORC-01: a unicidade (user_id, competence) é garantida por índice único
// PARCIAL (WHERE deleted_at IS NULL), criado em repository.EnsureGlobalBudgetIndexes
// — a tag uniqueIndex do GORM não expressa índice parcial, e um índice full
// impediria recriar o teto após um soft delete.
type GlobalBudget struct {
	ID           uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	UserID       uuid.UUID       `gorm:"type:uuid;not null" json:"user_id"`
	Competence   string          `gorm:"size:7;not null" json:"competence"`
	Ceiling      decimal.Decimal `gorm:"type:decimal(10,2);not null" json:"ceiling"`
	AutoAdjusted bool            `gorm:"not null;default:false" json:"auto_adjusted"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	DeletedAt    gorm.DeletedAt  `gorm:"index" json:"-"`
}

func (b *GlobalBudget) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}
