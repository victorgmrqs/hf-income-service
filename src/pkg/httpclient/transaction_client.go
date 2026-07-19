package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

var (
	// ErrUpstreamTimeout is returned when a call to hf-transaction-service exceeds the 5s per-call limit.
	ErrUpstreamTimeout = errors.New("upstream timeout")
	// ErrUpstreamError is returned when hf-transaction-service responds with an unexpected status code.
	ErrUpstreamError = errors.New("upstream error")
)

// ExpenseTotalsOutput holds the aggregated expense totals from GET /expenses/user/{id}/totals.
type ExpenseTotalsOutput struct {
	Competence    string          `json:"competence"`
	TotalPersonal decimal.Decimal `json:"total_personal"`
	TotalShared   decimal.Decimal `json:"total_shared"`
	TotalGeneral  decimal.Decimal `json:"total_general"`
}

// PendingBillOutput represents a single pending account payable from GET /accounts-payable.
type PendingBillOutput struct {
	ID          string          `json:"id"`
	Description string          `json:"description"`
	Amount      decimal.Decimal `json:"amount"`
	DueDate     string          `json:"due_date"`
	Status      string          `json:"status"`
	Recurrence  string          `json:"recurrence"`
	PaidAt      *string         `json:"paid_at"`
	ExpenseID   *string         `json:"expense_id"`
}

// ExpensesByCategoryOutput holds totals per category from GET /expenses/user/{id}/by-category.
type ExpensesByCategoryOutput struct {
	CategoryID string          `json:"category_id"`
	Total      decimal.Decimal `json:"total"`
}

// TransactionClient abstracts all calls to hf-transaction-service.
// Implementations must be injected via constructor — never instantiated inside use cases.
type TransactionClient interface {
	GetExpenseTotals(ctx context.Context, userID, competence string) (*ExpenseTotalsOutput, error)
	GetAccountsPayable(ctx context.Context, userID, dueDateUntil string) ([]PendingBillOutput, error)
	GetExpensesByCategory(ctx context.Context, userID, categoryID, competence string) (*ExpensesByCategoryOutput, error)
}

type transactionClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewTransactionClient creates a TransactionClient pointing at baseURL.
// Pass a custom httpClient for testing; nil uses a plain http.Client (no global timeout —
// per-call timeout of 5s is enforced inside each method via context.WithTimeout).
func NewTransactionClient(baseURL string, httpClient *http.Client) TransactionClient {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &transactionClient{baseURL: baseURL, httpClient: httpClient}
}

// envelope is the standard JSON wrapper used by hf-transaction-service responses.
type envelope[T any] struct {
	Data T `json:"data"`
}

// get executes a GET with a 5s per-call timeout and decodes the envelope into out.
func (c *transactionClient) get(ctx context.Context, path string, out any) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return ErrUpstreamTimeout
		}
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return ErrUpstreamTimeout
		}
		return fmt.Errorf("%w: %v", ErrUpstreamError, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%w: status %d", ErrUpstreamError, resp.StatusCode)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

func (c *transactionClient) GetExpenseTotals(ctx context.Context, userID, competence string) (*ExpenseTotalsOutput, error) {
	path := fmt.Sprintf("/api/v1/expenses/user/%s/totals?competence=%s", userID, competence)
	var env envelope[*ExpenseTotalsOutput]
	if err := c.get(ctx, path, &env); err != nil {
		return nil, err
	}
	if env.Data == nil {
		return nil, fmt.Errorf("%w: empty data for expense totals", ErrUpstreamError)
	}
	return env.Data, nil
}

func (c *transactionClient) GetAccountsPayable(ctx context.Context, userID, dueDateUntil string) ([]PendingBillOutput, error) {
	path := fmt.Sprintf("/api/v1/accounts-payable?user_id=%s&status=PENDING&due_date_until=%s", userID, dueDateUntil)
	var env envelope[[]PendingBillOutput]
	if err := c.get(ctx, path, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

func (c *transactionClient) GetExpensesByCategory(ctx context.Context, userID, categoryID, competence string) (*ExpensesByCategoryOutput, error) {
	path := fmt.Sprintf("/api/v1/expenses/user/%s/by-category?category_id=%s&competence=%s", userID, categoryID, competence)
	var env envelope[*ExpensesByCategoryOutput]
	if err := c.get(ctx, path, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// LastDayOfMonth returns the last calendar day of competence (format "YYYY-MM") as "YYYY-MM-DD".
// Used to build the due_date_until parameter for GetAccountsPayable.
func LastDayOfMonth(competence string) (string, error) {
	t, err := time.Parse("2006-01", competence)
	if err != nil {
		return "", fmt.Errorf("invalid competence format: %w", err)
	}
	// Day 0 of month+1 == last day of month; handles Feb leap years automatically.
	lastDay := time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	return lastDay.Format("2006-01-02"), nil
}
