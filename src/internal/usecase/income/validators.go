package income

import (
	"regexp"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
)

// competenceRe valida o formato YYYY-MM com mês entre 01 e 12 (REC-01).
var competenceRe = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

func isValidCompetence(competence string) bool {
	return competenceRe.MatchString(competence)
}

func isValidIncomeType(t entity.IncomeType) bool {
	switch t {
	case entity.IncomeTypeSalary,
		entity.IncomeTypeFreelance,
		entity.IncomeTypeInvestment,
		entity.IncomeTypeRental,
		entity.IncomeTypeOther:
		return true
	default:
		return false
	}
}
