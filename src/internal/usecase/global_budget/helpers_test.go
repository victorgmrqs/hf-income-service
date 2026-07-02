package global_budget

import (
	"log/slog"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/victorgmrqs/hf-income-service/src/pkg/observability"
)

func testLogger() *slog.Logger { return observability.NewLogger("test") }

func validCreateInput() CreateInput {
	return CreateInput{
		UserID:     uuid.New(),
		Competence: "2026-06",
		Ceiling:    decimal.RequireFromString("5000.00"),
	}
}
