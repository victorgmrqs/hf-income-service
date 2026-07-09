// Package httpclient fornece um cliente HTTP para o hf-transaction-service.
// O acesso é feito por interface (TransactionClient) para permitir mocks nos
// testes unitários dos use cases que dependem dele (ex.: auto-ajuste do teto,
// cálculo de saldo). Contrato dos endpoints: INTEGRATIONS.md.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/shopspring/decimal"
)

// TransactionClient abstrai as chamadas ao hf-transaction-service (INTEGRATIONS.md).
type TransactionClient interface {
	// GetExpenseTotals retorna os totais de despesas do usuário na competência
	// (consumido pelo auto-ajuste ORC-03/04 e pelo saldo SAL-01).
	GetExpenseTotals(ctx context.Context, userID, competence string) (*ExpenseTotalsOutput, error)
}

// ExpenseTotalsOutput é o payload de GET /api/v1/expenses/user/{id}/totals (INTEGRATIONS.md).
type ExpenseTotalsOutput struct {
	Competence    string          `json:"competence"`
	TotalPersonal decimal.Decimal `json:"total_personal"`
	TotalShared   decimal.Decimal `json:"total_shared"`
	TotalGeneral  decimal.Decimal `json:"total_general"`
}

// Client é a implementação concreta sobre net/http.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

var _ TransactionClient = (*Client)(nil)

// NewClient cria um cliente apontando para a base URL do hf-transaction-service.
// Timeout de 5s conforme INTEGRATIONS.md / FDD-002 §6 (resiliência).
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Get executa um GET em baseURL+path e decodifica o corpo JSON em out (quando != nil).
// Retorna erro em falha de transporte ou status fora da faixa 2xx.
func (c *Client) Get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call hf-transaction-service: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("hf-transaction-service returned status %d", resp.StatusCode)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// expenseTotalsEnvelope segue o envelope {data, error} padrão dos serviços Home Finance.
type expenseTotalsEnvelope struct {
	Data *ExpenseTotalsOutput `json:"data"`
}

// GetExpenseTotals busca os totais de despesas do usuário na competência:
// GET /api/v1/expenses/user/{user_id}/totals?competence=YYYY-MM (INTEGRATIONS.md).
func (c *Client) GetExpenseTotals(ctx context.Context, userID, competence string) (*ExpenseTotalsOutput, error) {
	path := fmt.Sprintf("/api/v1/expenses/user/%s/totals?competence=%s",
		url.PathEscape(userID), url.QueryEscape(competence))
	var env expenseTotalsEnvelope
	if err := c.Get(ctx, path, &env); err != nil {
		return nil, err
	}
	if env.Data == nil {
		return nil, fmt.Errorf("hf-transaction-service returned empty data for expense totals")
	}
	return env.Data, nil
}
