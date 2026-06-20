package income

import "errors"

// Erros de domínio do REC. Cada um referencia a regra de negócio que o origina
// (ver docs/rules/REC.md e o FDD docs/fdds/fdd-001-rec.md, seção 6).

// REC-01
var ErrMissingRequiredField = errors.New("required field is missing")

// REC-02
var ErrInvalidAmount = errors.New("amount must be greater than zero")

// REC-01
var ErrInvalidIncomeType = errors.New("income type is invalid")

// REC-01
var ErrInvalidCompetence = errors.New("competence must be in YYYY-MM format")

// REC-01
var ErrInvalidDate = errors.New("date must be in YYYY-MM-DD format")

var ErrIncomeNotFound = errors.New("income not found")

// REC-03
var ErrCannotEditPropagatedIncome = errors.New("propagated income cannot be edited")
