package income

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
)

// CreateInput são os dados de entrada para criar uma receita.
type CreateInput struct {
	UserID      uuid.UUID
	Description string
	Amount      decimal.Decimal
	Date        string // YYYY-MM-DD
	Competence  string // YYYY-MM
	Type        entity.IncomeType
	Recurrent   bool
}

// UpdateInput contém os campos editáveis de uma receita (REC-03).
// Campos imutáveis (user_id, origin_id, competence) não estão aqui.
type UpdateInput struct {
	ID          uuid.UUID
	Description *string
	Amount      *decimal.Decimal
	Date        *string
	Type        *entity.IncomeType
	Recurrent   *bool
}

// IncomeOutput é a representação pública de uma receita.
type IncomeOutput struct {
	ID          uuid.UUID       `json:"id"`
	UserID      uuid.UUID       `json:"user_id"`
	Description string          `json:"description"`
	Amount      decimal.Decimal `json:"amount"`
	Date        string          `json:"date"`
	Competence  string          `json:"competence"`
	Type        string          `json:"type"`
	Recurrent   bool            `json:"recurrent"`
	OriginID    *uuid.UUID      `json:"origin_id"`
	CreatedAt   time.Time       `json:"created_at"`
}

// ListOutput é a listagem por competência com o total agregado (REC-06).
type ListOutput struct {
	Competence  string          `json:"competence"`
	TotalIncome decimal.Decimal `json:"total_income"`
	Items       []IncomeOutput  `json:"items"`
}
