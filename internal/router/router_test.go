package router

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestMain(m *testing.M) {
	_ = logging.Init()
	os.Exit(m.Run())
}

func TestRegisterRoute(t *testing.T) {
	h := handler.NewUserHandler(stubUserStorage{})
	srv := httptest.NewServer(New(h))
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
	h := handler.NewUserHandler(stubUserStorage{})
	srv := httptest.NewServer(New(h))
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

func TestRegisterRouteNotFound(t *testing.T) {
	h := handler.NewUserHandler(stubUserStorage{})
	srv := httptest.NewServer(New(h))
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
