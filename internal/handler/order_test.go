package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kurochkin-evgeniy/gopher_mart/internal/auth"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/middleware"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/storage/postgres"
)

type mockOrderStorage struct {
	getOrderByNumber func(ctx context.Context, number string) (postgres.Order, int64, error)
	createOrder      func(ctx context.Context, userID int64, number string) error
	listOrdersByUser func(ctx context.Context, userID int64) ([]postgres.Order, error)
}

func (m *mockOrderStorage) GetOrderByNumber(ctx context.Context, number string) (postgres.Order, int64, error) {
	return m.getOrderByNumber(ctx, number)
}

func (m *mockOrderStorage) CreateOrder(ctx context.Context, userID int64, number string) error {
	return m.createOrder(ctx, userID, number)
}

func (m *mockOrderStorage) ListOrdersByUser(ctx context.Context, userID int64) ([]postgres.Order, error) {
	return m.listOrdersByUser(ctx, userID)
}

func withUserContext(userID int64) context.Context {
	return context.WithValue(context.Background(), middleware.UserIDKey, userID)
}

func TestUploadAccepted(t *testing.T) {
	storage := &mockOrderStorage{
		getOrderByNumber: func(ctx context.Context, number string) (postgres.Order, int64, error) {
			return postgres.Order{}, 0, postgres.ErrOrderNotFound
		},
		createOrder: func(ctx context.Context, userID int64, number string) error {
			if userID != 1 || number != "12345678903" {
				t.Fatalf("unexpected create: user=%d number=%q", userID, number)
			}
			return nil
		},
	}

	h := NewOrderHandler(storage)
	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader("12345678903\n"))
	req = req.WithContext(withUserContext(1))
	rec := httptest.NewRecorder()

	h.Upload(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status: got %d", rec.Code)
	}
}

func TestUploadAlreadyUploadedByUser(t *testing.T) {
	storage := &mockOrderStorage{
		getOrderByNumber: func(ctx context.Context, number string) (postgres.Order, int64, error) {
			return postgres.Order{Number: number, Status: "NEW"}, 1, nil
		},
	}

	h := NewOrderHandler(storage)
	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader("12345678903"))
	req = req.WithContext(withUserContext(1))
	rec := httptest.NewRecorder()

	h.Upload(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d", rec.Code)
	}
}

func TestUploadConflict(t *testing.T) {
	storage := &mockOrderStorage{
		getOrderByNumber: func(ctx context.Context, number string) (postgres.Order, int64, error) {
			return postgres.Order{Number: number}, 2, nil
		},
	}

	h := NewOrderHandler(storage)
	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader("12345678903"))
	req = req.WithContext(withUserContext(1))
	rec := httptest.NewRecorder()

	h.Upload(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status: got %d", rec.Code)
	}
}

func TestUploadUnauthorized(t *testing.T) {
	h := NewOrderHandler(&mockOrderStorage{})
	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader("12345678903"))
	rec := httptest.NewRecorder()

	h.Upload(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d", rec.Code)
	}
}

func TestUploadUnprocessable(t *testing.T) {
	h := NewOrderHandler(&mockOrderStorage{})
	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader("12345678900"))
	req = req.WithContext(withUserContext(1))
	rec := httptest.NewRecorder()

	h.Upload(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status: got %d", rec.Code)
	}
}

func TestUploadBadRequest(t *testing.T) {
	h := NewOrderHandler(&mockOrderStorage{})
	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader(""))
	req = req.WithContext(withUserContext(1))
	rec := httptest.NewRecorder()

	h.Upload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d", rec.Code)
	}
}

func TestListOrders(t *testing.T) {
	uploadedAt := time.Date(2020, 12, 10, 15, 15, 45, 0, time.FixedZone("MSK", 3*3600))
	accrual := 500.0

	storage := &mockOrderStorage{
		listOrdersByUser: func(ctx context.Context, userID int64) ([]postgres.Order, error) {
			return []postgres.Order{
				{
					Number:     "9278923470",
					Status:     "PROCESSED",
					Accrual:    &accrual,
					UploadedAt: uploadedAt,
				},
			}, nil
		},
	}

	h := NewOrderHandler(storage)
	req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	req = req.WithContext(withUserContext(1))
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, body %q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"number":"9278923470"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"accrual":500`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestListOrdersNoContent(t *testing.T) {
	storage := &mockOrderStorage{
		listOrdersByUser: func(ctx context.Context, userID int64) ([]postgres.Order, error) {
			return nil, nil
		},
	}

	h := NewOrderHandler(storage)
	req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	req = req.WithContext(withUserContext(1))
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: got %d", rec.Code)
	}
}

func TestListOrdersUnauthorized(t *testing.T) {
	h := NewOrderHandler(&mockOrderStorage{})
	req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d", rec.Code)
	}
}

func TestListOrdersStorageError(t *testing.T) {
	storage := &mockOrderStorage{
		listOrdersByUser: func(ctx context.Context, userID int64) ([]postgres.Order, error) {
			return nil, errors.New("db down")
		},
	}

	h := NewOrderHandler(storage)
	req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	req = req.WithContext(withUserContext(1))
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d", rec.Code)
	}
}

func TestUploadWithAuthHeaderIntegration(t *testing.T) {
	token, err := auth.GenerateToken(3)
	if err != nil {
		t.Fatal(err)
	}

	storage := &mockOrderStorage{
		getOrderByNumber: func(ctx context.Context, number string) (postgres.Order, int64, error) {
			return postgres.Order{}, 0, postgres.ErrOrderNotFound
		},
		createOrder: func(ctx context.Context, userID int64, number string) error {
			if userID != 3 {
				t.Fatalf("user id: got %d", userID)
			}
			return nil
		},
	}

	h := NewOrderHandler(storage)
	handler := middleware.Authenticate(http.HandlerFunc(h.Upload))

	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("12345678903"))
	req.Header.Set(auth.AuthHeader, auth.AuthScheme+" "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status: got %d", rec.Code)
	}
}
