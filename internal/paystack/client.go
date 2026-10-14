package paystack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type InitResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		AccessCode       string `json:"access_code"`
		Amount           int64  `json:"amount"`
		AuthorizationURL string `json:"authorization_url"`
		Reference        string `json:"reference"`
		Status           string `json:"status"`
	} `json:"data"`
}

// VerifyResponse is what GET /transaction/verify/<reference> returns. The outer Status only says the
// API call worked; Data.Status says how the payment went: "success", "failed", "abandoned", "ongoing", ...
type VerifyResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Status    string `json:"status"`
		Reference string `json:"reference"`
		Amount    int64  `json:"amount"` // in pesewas
		Currency  string `json:"currency"`
	} `json:"data"`
}

type initRequest struct {
	Amount    int64  `json:"amount"` // In pesewas
	Email     string `json:"email"`
	Reference string `json:"reference"`
}
type Client struct {
	secretKey  string
	baseURL    string
	httpClient *http.Client
}

func NewClient(secretKey string) *Client {
	return &Client{
		secretKey:  secretKey,
		baseURL:    "https://api.paystack.co",
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) InitializeTransaction(ctx context.Context, amount int64, email, reference string) (authURL string, err error) {
	url := fmt.Sprintf("%s/transaction/initialize", c.baseURL)
	body, err := json.Marshal(initRequest{
		Amount:    amount,
		Email:     email,
		Reference: reference,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.secretKey))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("client error: failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var envelope InitResponse

	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return "", fmt.Errorf("decode response (status %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK || !envelope.Status {
		return "", fmt.Errorf("paystack initialize failed (status %d): %s", resp.StatusCode, envelope.Message)
	}
	return envelope.Data.AuthorizationURL, nil
}

// GET /transaction/verify/<reference>
func (c *Client) VerifyTransaction(ctx context.Context, reference string) (*VerifyResponse, error) {
	url := fmt.Sprintf("%s/transaction/verify/%s", c.baseURL, reference)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("(VerifyTransaction): build request failed: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.secretKey))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("(VerifyTransaction): request send failed: %w", err)
	}
	defer resp.Body.Close()

	var out VerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("(VerifyTransaction): response decode failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK || !out.Status {
		return nil, fmt.Errorf("paystack verification failed: (status %d): %s", resp.StatusCode, out.Message)
	}
	return &out, nil
}
