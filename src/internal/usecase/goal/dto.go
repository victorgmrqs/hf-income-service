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

// ComparisonInput identifica o comparativo mensal de metas do usuário (MET-05/07).
type ComparisonInput struct {
	UserID     uuid.UUID
	Competence string // YYYY-MM
}

// ComparisonItemOutput é a linha do comparativo por categoria (FDD-003 §5).
// Campos ponteiro degradam para null quando o hf-transaction-service está
// indisponível ou quando não há base de comparação.
type ComparisonItemOutput struct {
	CategoryID          uuid.UUID        `json:"category_id"`
	CategoryName        *string          `json:"category_name"`
	PreviousMonthAmount *decimal.Decimal `json:"previous_month_amount"`
	CurrentMonthAmount  *decimal.Decimal `json:"current_month_amount"`
	TargetAmount        decimal.Decimal  `json:"target_amount"`
	OnTrack             *bool            `json:"on_track"`
	VariationPct        *float64         `json:"variation_pct"`
	VariationLabel      *string          `json:"variation_label"`
	TargetProgressPct   *float64         `json:"target_progress_pct"`
}

// CloseMonthInput identifica a competência a fechar (MET-06).
type CloseMonthInput struct {
	Competence string // YYYY-MM
}

// CloseMonthOutput é o resultado do fechamento: quantidade de metas fechadas.
type CloseMonthOutput struct {
	Closed int `json:"closed"`
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
