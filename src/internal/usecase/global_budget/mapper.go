package global_budget

import "github.com/victorgmrqs/hf-income-service/src/internal/entity"

func toOutput(b *entity.GlobalBudget) GlobalBudgetOutput {
	return GlobalBudgetOutput{
		ID:           b.ID,
		UserID:       b.UserID,
		Competence:   b.Competence,
		Ceiling:      b.Ceiling,
		AutoAdjusted: b.AutoAdjusted,
		CreatedAt:    b.CreatedAt,
	}
}
