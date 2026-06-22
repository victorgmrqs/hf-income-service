package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type IncomeType string

const (
	IncomeTypeSalary     IncomeType = "SALARY"
	IncomeTypeFreelance  IncomeType = "FREELANCE"
	IncomeTypeInvestment IncomeType = "INVESTMENT"
	IncomeTypeRental     IncomeType = "RENTAL"
	IncomeTypeOther      IncomeType = "OTHER"
)

// Income representa uma receita mensal de um usuário (REC).
// origin_id é preenchido apenas em registros propagados (REC-03/REC-04) e,
// quando presente, torna o registro imutável (não editável).
type Income struct {
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID       `gorm:"type:uuid;not null;index:idx_incomes_user_competence,priority:1" json:"user_id"`
	Description string          `gorm:"size:200;not null" json:"description"`
	Amount      decimal.Decimal `gorm:"type:decimal(12,2);not null" json:"amount"`
	Date        time.Time       `gorm:"not null" json:"date"`
	Competence  string          `gorm:"size:7;not null;index:idx_incomes_user_competence,priority:2;uniqueIndex:idx_incomes_origin_competence,priority:2" json:"competence"`
	Type        IncomeType      `gorm:"size:20;not null" json:"type"`
	Recurrent   bool            `gorm:"not null;default:false" json:"recurrent"`
	// OriginID + Competence formam um índice único (idempotência da propagação, REC-04).
	// No PostgreSQL múltiplas linhas com origin_id NULL coexistem (NULLs são distintos).
	OriginID  *uuid.UUID     `gorm:"type:uuid;index;uniqueIndex:idx_incomes_origin_competence,priority:1" json:"origin_id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (income *Income) BeforeCreate(tx *gorm.DB) (err error) {
	if income.ID == uuid.Nil {
		income.ID = uuid.New()
	}
	return nil
}
