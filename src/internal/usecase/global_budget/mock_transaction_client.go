package global_budget

import (
	"context"

	"github.com/victorgmrqs/hf-income-service/src/pkg/httpclient"
)

// mockTransactionClient é um mock manual de httpclient.TransactionClient para
// os testes unitários dos use cases que consomem o hf-transaction-service.
type mockTransactionClient struct {
	GetExpenseTotalsFn func(ctx context.Context, userID, competence string) (*httpclient.ExpenseTotalsOutput, error)
}

var _ httpclient.TransactionClient = (*mockTransactionClient)(nil)

func (m *mockTransactionClient) GetExpenseTotals(ctx context.Context, userID, competence string) (*httpclient.ExpenseTotalsOutput, error) {
	return m.GetExpenseTotalsFn(ctx, userID, competence)
}
