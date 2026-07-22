package goal

import "github.com/victorgmrqs/hf-income-service/src/internal/entity"

func toOutput(g *entity.ReductionGoal) GoalOutput {
	return GoalOutput{
		ID:             g.ID,
		UserID:         g.UserID,
		CategoryID:     g.CategoryID,
		Competence:     g.Competence,
		TargetAmount:   g.TargetAmount,
		PreviousAmount: g.PreviousAmount,
		Achieved:       g.Achieved,
		CreatedAt:      g.CreatedAt,
	}
}

func toOutputs(goals []entity.ReductionGoal) []GoalOutput {
	outputs := make([]GoalOutput, len(goals))
	for i := range goals {
		outputs[i] = toOutput(&goals[i])
	}
	return outputs
}
