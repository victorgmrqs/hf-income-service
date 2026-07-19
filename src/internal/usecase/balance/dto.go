package balance

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// GetInput identifica o saldo pela competência do usuário (SAL-01).
type GetInput struct {
	UserID     uuid.UUID
	Competence string // YYYY-MM
}

// BalanceOutput é o saldo mensal calculado sob demanda — sem persistência
// (FDD-004 §5). Ceiling e CeilingUsagePct são ponteiros: null quando o usuário
// não tem teto cadastrado na competência (ORC-06/07).
type BalanceOutput struct {
	UserID              uuid.UUID        `json:"user_id"`
	Competence          string           `json:"competence"`
	TotalIncome         decimal.Decimal  `json:"total_income"`
	TotalPersonal       decimal.Decimal  `json:"total_personal"`
	TotalShared         decimal.Decimal  `json:"total_shared"`
	TotalExpenses       decimal.Decimal  `json:"total_expenses"`
	BalanceToday        decimal.Decimal  `json:"balance_today"`
	CommittedBills      decimal.Decimal  `json:"committed_bills"`
	ProjectedBalance    decimal.Decimal  `json:"projected_balance"`
	IsProjectedNegative bool             `json:"is_projected_negative"`
	Ceiling             *decimal.Decimal `json:"ceiling"`
	CeilingUsagePct     *int             `json:"ceiling_usage_pct"`
	CeilingExceeded     bool             `json:"ceiling_exceeded"`
}
