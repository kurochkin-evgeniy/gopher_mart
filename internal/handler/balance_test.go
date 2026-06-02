package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kurochkin-evgeniy/gopher_mart/internal/model"
)

type mockBalanceStorage struct {
	getBalance       func(ctx context.Context, userID int64) (model.Balance, error)
	withdraw         func(ctx context.Context, userID int64, orderNumber string, sum float64) error
	listWithdrawals  func(ctx context.Context, userID int64) ([]model.Withdrawal, error)
}

func (m *mockBalanceStorage) GetBalance(ctx context.Context, userID int64) (model.Balance, error) {
	return m.getBalance(ctx, userID)
}

func (m *mockBalanceStorage) Withdraw(ctx context.Context, userID int64, orderNumber string, sum float64) error {
	return m.withdraw(ctx, userID, orderNumber, sum)
}

func (m *mockBalanceStorage) ListWithdrawals(ctx context.Context, userID int64) ([]model.Withdrawal, error) {
	return m.listWithdrawals(ctx, userID)
}

func TestGetBalance(t *testing.T) {
	storage := &mockBalanceStorage{
		getBalance: func(ctx context.Context, userID int64) (model.Balance, error) {
			return model.Balance{Current: 500.5, Withdrawn: 42}, nil
		},
	}

	h := NewBalanceHandler(storage)
	req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
	req = req.WithContext(withUserContext(1))
	rec := httptest.NewRecorder()

	h.GetBalance(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"current":500.5`) {
		t.Fatalf("body: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"withdrawn":42`) {
		t.Fatalf("body: %s", rec.Body.String())
	}
}

func TestWithdrawSuccess(t *testing.T) {
	storage := &mockBalanceStorage{
		withdraw: func(ctx context.Context, userID int64, orderNumber string, sum float64) error {
			if userID != 1 || orderNumber != "79927398713" || sum != 751 {
				t.Fatalf("unexpected withdraw args")
			}
			return nil
		},
	}

	h := NewBalanceHandler(storage)
	req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", strings.NewReader(`{"order":"79927398713","sum":751}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserContext(1))
	rec := httptest.NewRecorder()

	h.Withdraw(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, body %q", rec.Code, rec.Body.String())
	}
}

func TestWithdrawInsufficientFunds(t *testing.T) {
	storage := &mockBalanceStorage{
		withdraw: func(ctx context.Context, userID int64, orderNumber string, sum float64) error {
			return model.ErrInsufficientFunds
		},
	}

	h := NewBalanceHandler(storage)
	req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", strings.NewReader(`{"order":"79927398713","sum":751}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserContext(1))
	rec := httptest.NewRecorder()

	h.Withdraw(rec, req)

	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("status: got %d", rec.Code)
	}
}

func TestWithdrawUnprocessable(t *testing.T) {
	h := NewBalanceHandler(&mockBalanceStorage{})
	req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", strings.NewReader(`{"order":"12345678900","sum":10}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserContext(1))
	rec := httptest.NewRecorder()

	h.Withdraw(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status: got %d", rec.Code)
	}
}

func TestListWithdrawals(t *testing.T) {
	processedAt := time.Date(2020, 12, 9, 16, 9, 57, 0, time.FixedZone("MSK", 3*3600))
	storage := &mockBalanceStorage{
		listWithdrawals: func(ctx context.Context, userID int64) ([]model.Withdrawal, error) {
			return []model.Withdrawal{
				{Order: "2377225624", Sum: 500, ProcessedAt: processedAt},
			}, nil
		},
	}

	h := NewBalanceHandler(storage)
	req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
	req = req.WithContext(withUserContext(1))
	rec := httptest.NewRecorder()

	h.ListWithdrawals(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"order":"2377225624"`) {
		t.Fatalf("body: %s", rec.Body.String())
	}
}

func TestListWithdrawalsNoContent(t *testing.T) {
	storage := &mockBalanceStorage{
		listWithdrawals: func(ctx context.Context, userID int64) ([]model.Withdrawal, error) {
			return nil, nil
		},
	}

	h := NewBalanceHandler(storage)
	req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
	req = req.WithContext(withUserContext(1))
	rec := httptest.NewRecorder()

	h.ListWithdrawals(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: got %d", rec.Code)
	}
}

func TestBalanceUnauthorized(t *testing.T) {
	h := NewBalanceHandler(&mockBalanceStorage{})

	rec := httptest.NewRecorder()
	h.GetBalance(rec, httptest.NewRequest(http.MethodGet, "/api/user/balance", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("get balance status: got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.Withdraw(rec, httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("withdraw status: got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ListWithdrawals(rec, httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("list withdrawals status: got %d", rec.Code)
	}
}

func TestGetBalanceStorageError(t *testing.T) {
	storage := &mockBalanceStorage{
		getBalance: func(ctx context.Context, userID int64) (model.Balance, error) {
			return model.Balance{}, errors.New("db down")
		},
	}

	h := NewBalanceHandler(storage)
	req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
	req = req.WithContext(withUserContext(1))
	rec := httptest.NewRecorder()

	h.GetBalance(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d", rec.Code)
	}
}
