package global_budget

import "errors"

// Erros de domínio do ORC. Cada um referencia a regra de negócio que o origina
// (ver docs/rules/ORC.md e o FDD docs/fdds/fdd-002-orc.md, seção 6).

// ORC-01
var ErrMissingRequiredField = errors.New("required field is missing")

// ORC-01
var ErrInvalidCeiling = errors.New("ceiling must be greater than zero")

// ORC-01
var ErrInvalidCompetence = errors.New("competence must be in YYYY-MM format")

// ORC-01
var ErrBudgetAlreadyExists = errors.New("a global budget already exists for this user and competence")

var ErrBudgetNotFound = errors.New("global budget not found")

// ORC-03
var ErrNoPreviousBudget = errors.New("no previous budget found to base the auto-adjustment on")

var ErrUpstreamUnavailable = errors.New("could not fetch expense data from hf-transaction-service")
