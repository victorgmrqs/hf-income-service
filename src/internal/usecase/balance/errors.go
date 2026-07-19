package balance

import "errors"

// Erros de domínio do SAL. Cada um referencia a regra de negócio que o origina
// (ver docs/rules/SAL.md e o FDD docs/fdds/fdd-004-sal.md, seção 6).

// SAL-01
var ErrMissingRequiredField = errors.New("required field is missing")

// SAL-01
var ErrInvalidCompetence = errors.New("competence must be in YYYY-MM format")

var ErrUpstreamTimeout = errors.New("expense service unavailable, please retry")

var ErrUpstreamError = errors.New("unexpected response from expense service")
