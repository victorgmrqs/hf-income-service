package global_budget

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CreateInput são os dados de entrada para criar um teto global (ORC).
type CreateInput struct {
	UserID     uuid.UUID
	Competence string // YYYY-MM
	Ceiling    decimal.Decimal
}

// GetInput identifica o teto pela competência do usuário (ORC-01).
type GetInput struct {
	UserID     uuid.UUID
	Competence string // YYYY-MM
}

// UpdateInput contém os campos editáveis de um teto global.
// A edição manual sempre força auto_adjusted = false (ORC-05).
type UpdateInput struct {
	ID      uuid.UUID
	Ceiling decimal.Decimal
}

// GlobalBudgetOutput é a representação pública de um teto global.
type GlobalBudgetOutput struct {
	ID           uuid.UUID       `json:"id"`
	UserID       uuid.UUID       `json:"user_id"`
	Competence   string          `json:"competence"`
	Ceiling      decimal.Decimal `json:"ceiling"`
	AutoAdjusted bool            `json:"auto_adjusted"`
	CreatedAt    time.Time       `json:"created_at"`
}
