package global_budget

import "regexp"

// competenceRe valida o formato YYYY-MM com mês entre 01 e 12 (ORC-01).
var competenceRe = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

func isValidCompetence(competence string) bool {
	return competenceRe.MatchString(competence)
}
