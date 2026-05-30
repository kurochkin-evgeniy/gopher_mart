package router

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/kurochkin-evgeniy/gopher_mart/internal/auth"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/handler"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/logging"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/storage/postgres"
)

type stubUserStorage struct{}

func (stubUserStorage) CreateUser(ctx context.Context, login, passwordHash string) (int64, error) {
	return 1, nil
}

func (stubUserStorage) GetUserByLogin(ctx context.Context, login string) (int64, string, error) {
	return 0, "", postgres.ErrUserNotFound
}

type stubOrderStorage struct{}

func (stubOrderStorage) GetOrderByNumber(ctx context.Context, number string) (postgres.Order, int64, error) {
	return postgres.Order{}, 0, postgres.ErrOrderNotFound
}

func (stubOrderStorage) CreateOrder(ctx context.Context, userID int64, number string) error {
	return nil
}

func (stubOrderStorage) ListOrdersByUser(ctx context.Context, userID int64) ([]postgres.Order, error) {
	return nil, nil
}

type stubBalanceStorage struct{}

func (stubBalanceStorage) GetBalance(ctx context.Context, userID int64) (postgres.Balance, error) {
	return postgres.Balance{}, nil
}

func (stubBalanceStorage) Withdraw(ctx context.Context, userID int64, orderNumber string, sum float64) error {
	return nil
}

func (stubBalanceStorage) ListWithdrawals(ctx context.Context, userID int64) ([]postgres.Withdrawal, error) {
	return nil, nil
}

func newTestRouter() http.Handler {
	userHandler := handler.NewUserHandler(stubUserStorage{})
	orderHandler := handler.NewOrderHandler(stubOrderStorage{})
	balanceHandler := handler.NewBalanceHandler(stubBalanceStorage{})
	return New(userHandler, orderHandler, balanceHandler)
}

func TestMain(m *testing.M) {
	_ = logging.Init()
	os.Exit(m.Run())
}

func TestRegisterRoute(t *testing.T) {
	srv := httptest.NewServer(newTestRouter())
	t.Cleanup(srv.Close)

	body := bytes.NewBufferString(`{"login":"router_user","password":"secret"}`)
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/user/register", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}

	authHeader := resp.Header.Get(auth.AuthHeader)
	if authHeader == "" {
		t.Fatal("expected Authorization header")
	}
}

func TestLoginRoute(t *testing.T) {
	srv := httptest.NewServer(newTestRouter())
	t.Cleanup(srv.Close)

	resp, err := http.Post(
		srv.URL+"/api/user/login",
		"application/json",
		bytes.NewBufferString(`{"login":"user","password":"secret"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestUploadOrderRoute(t *testing.T) {
	srv := httptest.NewServer(newTestRouter())
	t.Cleanup(srv.Close)

	token, err := auth.GenerateToken(1)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/user/orders", strings.NewReader("12345678903"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(auth.AuthHeader, auth.AuthScheme+" "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestUploadOrderUnauthorized(t *testing.T) {
	srv := httptest.NewServer(newTestRouter())
	t.Cleanup(srv.Close)

	resp, err := http.Post(srv.URL+"/api/user/orders", "text/plain", strings.NewReader("12345678903"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestGetBalanceRoute(t *testing.T) {
	srv := httptest.NewServer(newTestRouter())
	t.Cleanup(srv.Close)

	token, err := auth.GenerateToken(1)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/user/balance", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(auth.AuthHeader, auth.AuthScheme+" "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestListOrdersNoContent(t *testing.T) {
	srv := httptest.NewServer(newTestRouter())
	t.Cleanup(srv.Close)

	token, err := auth.GenerateToken(1)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/user/orders", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(auth.AuthHeader, auth.AuthScheme+" "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestRegisterRouteNotFound(t *testing.T) {
	srv := httptest.NewServer(newTestRouter())
	t.Cleanup(srv.Close)

	resp, err := http.Post(srv.URL+"/api/user/unknown", "application/json", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}
