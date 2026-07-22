package goal

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CreateInput são os dados de entrada para criar uma meta de redução (MET).
type CreateInput struct {
	UserID       uuid.UUID
	CategoryID   uuid.UUID
	Competence   string // YYYY-MM
	TargetAmount decimal.Decimal
}

// ListInput identifica as metas de um usuário na competência.
type ListInput struct {
	UserID     uuid.UUID
	Competence string // YYYY-MM
}

// UpdateInput contém o único campo editável de uma meta (MET-04).
type UpdateInput struct {
	ID           uuid.UUID
	TargetAmount decimal.Decimal
}

// GoalOutput é a representação pública de uma meta de redução.
type GoalOutput struct {
	ID             uuid.UUID        `json:"id"`
	UserID         uuid.UUID        `json:"user_id"`
	CategoryID     uuid.UUID        `json:"category_id"`
	Competence     string           `json:"competence"`
	TargetAmount   decimal.Decimal  `json:"target_amount"`
	PreviousAmount *decimal.Decimal `json:"previous_amount"`
	Achieved       *bool            `json:"achieved"`
	CreatedAt      time.Time        `json:"created_at"`
}
