package income

import "github.com/victorgmrqs/hf-income-service/src/internal/entity"

func toOutput(i *entity.Income) IncomeOutput {
	return IncomeOutput{
		ID:          i.ID,
		UserID:      i.UserID,
		Description: i.Description,
		Amount:      i.Amount,
		Date:        i.Date.Format(dateLayout),
		Competence:  i.Competence,
		Type:        string(i.Type),
		Recurrent:   i.Recurrent,
		OriginID:    i.OriginID,
		CreatedAt:   i.CreatedAt,
	}
}

func toOutputs(items []entity.Income) []IncomeOutput {
	out := make([]IncomeOutput, 0, len(items))
	for idx := range items {
		out = append(out, toOutput(&items[idx]))
	}
	return out
}
