package income

import (
	"log/slog"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/victorgmrqs/hf-income-service/src/internal/entity"
	"github.com/victorgmrqs/hf-income-service/src/pkg/observability"
)

func testLogger() *slog.Logger { return observability.NewLogger("test") }

func validCreateInput() CreateInput {
	return CreateInput{
		UserID:      uuid.New(),
		Description: "Salário junho",
		Amount:      decimal.RequireFromString("5000.00"),
		Date:        "2026-06-05",
		Competence:  "2026-06",
		Type:        entity.IncomeTypeSalary,
		Recurrent:   false,
	}
}
