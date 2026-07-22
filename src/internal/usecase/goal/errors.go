package goal

import "errors"

// Erros de domínio do MET. Cada um referencia a regra de negócio que o origina
// (ver docs/rules/MET.md e o FDD docs/fdds/fdd-003-met.md, seção 6).

// MET-04
var ErrMissingRequiredField = errors.New("required field is missing")

// MET-04
var ErrInvalidTargetAmount = errors.New("target amount must be greater than zero")

// MET-04
var ErrInvalidCompetence = errors.New("competence must be in YYYY-MM format")

// MET-04
var ErrGoalAlreadyExists = errors.New("a reduction goal already exists for this user, category and competence")

var ErrGoalNotFound = errors.New("reduction goal not found")
