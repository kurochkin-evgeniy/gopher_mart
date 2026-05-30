// Package accrual предоставляет клиент для системы расчёта начислений.
package accrual

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ErrNotRegistered возвращается, когда заказ не зарегистрирован в системе начислений.
var ErrNotRegistered = fmt.Errorf("order not registered")

// ErrTooManyRequests возвращается при превышении лимита запросов к системе начислений.
type ErrTooManyRequests struct {
	RetryAfter time.Duration
}

func (e ErrTooManyRequests) Error() string {
	return fmt.Sprintf("too many requests, retry after %s", e.RetryAfter)
}

// OrderInfo содержит информацию о расчёте начисления по заказу.
type OrderInfo struct {
	Order   string   `json:"order"`
	Status  string   `json:"status"`
	Accrual *float64 `json:"accrual,omitempty"`
}

// Client выполняет HTTP-запросы к системе расчёта начислений.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient создаёт клиент системы начислений.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetOrder запрашивает информацию о начислении по номеру заказа.
func (c *Client) GetOrder(ctx context.Context, number string) (*OrderInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/orders/"+number, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var info OrderInfo
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
		return &info, nil
	case http.StatusNoContent:
		return nil, ErrNotRegistered
	case http.StatusTooManyRequests:
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		return nil, ErrTooManyRequests{RetryAfter: retryAfter}
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}

func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return time.Minute
	}

	seconds, err := strconv.Atoi(value)
	if err == nil {
		return time.Duration(seconds) * time.Second
	}

	if t, err := http.ParseTime(value); err == nil {
		return time.Until(t)
	}

	return time.Minute
}

// MapStatus преобразует статус системы начислений в статус заказа в «Гофермарте».
func MapStatus(status string) string {
	switch status {
	case "INVALID":
		return "INVALID"
	case "PROCESSED":
		return "PROCESSED"
	default:
		return "PROCESSING"
	}
}
