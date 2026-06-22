// Package httpclient fornece um cliente HTTP de baixo nível para o hf-transaction-service.
// Use TransactionClient (definida em transaction_client.go) para injeção nos use cases.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client é o transporte HTTP de baixo nível sobre net/http.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient cria um cliente apontando para a base URL do hf-transaction-service.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
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
