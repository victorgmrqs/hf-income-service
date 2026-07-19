package global_budget

import (
	"context"

	"github.com/victorgmrqs/hf-income-service/src/pkg/httpclient"
)

// mockTransactionClient é um mock manual de httpclient.TransactionClient para
// os testes unitários dos use cases que consomem o hf-transaction-service.
// Os métodos não usados pelo ORC retornam zero-values quando o Fn não é definido.
type mockTransactionClient struct {
	GetExpenseTotalsFn      func(ctx context.Context, userID, competence string) (*httpclient.ExpenseTotalsOutput, error)
	GetAccountsPayableFn    func(ctx context.Context, userID, dueDateUntil string) ([]httpclient.PendingBillOutput, error)
	GetExpensesByCategoryFn func(ctx context.Context, userID, categoryID, competence string) (*httpclient.ExpensesByCategoryOutput, error)
}

var _ httpclient.TransactionClient = (*mockTransactionClient)(nil)

func (m *mockTransactionClient) GetExpenseTotals(ctx context.Context, userID, competence string) (*httpclient.ExpenseTotalsOutput, error) {
	return m.GetExpenseTotalsFn(ctx, userID, competence)
}

func (m *mockTransactionClient) GetAccountsPayable(ctx context.Context, userID, dueDateUntil string) ([]httpclient.PendingBillOutput, error) {
	if m.GetAccountsPayableFn == nil {
		return nil, nil
	}
	return m.GetAccountsPayableFn(ctx, userID, dueDateUntil)
}

func (m *mockTransactionClient) GetExpensesByCategory(ctx context.Context, userID, categoryID, competence string) (*httpclient.ExpensesByCategoryOutput, error) {
	if m.GetExpensesByCategoryFn == nil {
		return nil, nil
	}
	return m.GetExpensesByCategoryFn(ctx, userID, categoryID, competence)
}
