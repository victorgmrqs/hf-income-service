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

// AutoAdjustInput identifica a competência de DESTINO do auto-ajuste (ORC-03/04).
// Ex.: Competence "2026-07" ajusta o teto de julho com base no gasto de junho.
type AutoAdjustInput struct {
	UserID     uuid.UUID
	Competence string // YYYY-MM
}

// PreviewInput identifica o usuário do preview; a competência corrente vem do relógio.
type PreviewInput struct {
	UserID uuid.UUID
}

// Valores de adjustment_reason no preview do auto-ajuste (ORC-03/04).
const (
	AdjustmentReasonSpendingBelowCeiling  = "spending_below_ceiling"
	AdjustmentReasonSpendingEqualsCeiling = "spending_equals_ceiling"
)

// PreviewOutput é o cálculo do auto-ajuste sem persistência (ORC-03/04, FDD §5).
type PreviewOutput struct {
	CurrentCompetence string          `json:"current_competence"`
	CurrentCeiling    decimal.Decimal `json:"current_ceiling"`
	CurrentSpending   decimal.Decimal `json:"current_spending"`
	NextCompetence    string          `json:"next_competence"`
	SuggestedCeiling  decimal.Decimal `json:"suggested_ceiling"`
	AdjustmentReason  string          `json:"adjustment_reason"`
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
