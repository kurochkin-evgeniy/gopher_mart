package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/kurochkin-evgeniy/gopher_mart/internal/auth"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/logging"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/storage/postgres"
)

type mockUserStorage struct {
	createUser func(ctx context.Context, login, passwordHash string) (int64, error)
}

func (m *mockUserStorage) CreateUser(ctx context.Context, login, passwordHash string) (int64, error) {
	return m.createUser(ctx, login, passwordHash)
}

func TestMain(m *testing.M) {
	_ = logging.Init()
	os.Exit(m.Run())
}

func TestDecodeJSON(t *testing.T) {
	var cred credentials
	if err := decodeJSON(strings.NewReader(`{"login":"a","password":"b"}`), &cred); err != nil {
		t.Fatal(err)
	}
	if cred.Login != "a" || cred.Password != "b" {
		t.Fatalf("unexpected credentials: %+v", cred)
	}

	err := decodeJSON(strings.NewReader(`{"login":"a","extra":1}`), &cred)
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestRegisterSuccess(t *testing.T) {
	storage := &mockUserStorage{
		createUser: func(ctx context.Context, login, passwordHash string) (int64, error) {
			if login != "user" || passwordHash == "" {
				t.Fatalf("unexpected create user args: login=%q hash empty=%v", login, passwordHash == "")
			}
			return 7, nil
		},
	}

	h := NewUserHandler(storage)
	body := bytes.NewBufferString(`{"login":"user","password":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, body %q", rec.Code, rec.Body.String())
	}

	authHeader := rec.Header().Get(auth.AuthHeader)
	if !strings.HasPrefix(authHeader, auth.AuthScheme+" ") {
		t.Fatalf("authorization header: %q", authHeader)
	}

	token := strings.TrimPrefix(authHeader, auth.AuthScheme+" ")
	claims, err := auth.ParseToken(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if claims.UserID != 7 {
		t.Fatalf("user id in token: got %d", claims.UserID)
	}
}

func TestRegisterLoginTaken(t *testing.T) {
	storage := &mockUserStorage{
		createUser: func(ctx context.Context, login, passwordHash string) (int64, error) {
			return 0, postgres.ErrLoginTaken
		},
	}

	h := NewUserHandler(storage)
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(`{"login":"user","password":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status: got %d", rec.Code)
	}
}

func TestRegisterBadRequest(t *testing.T) {
	h := NewUserHandler(&mockUserStorage{})

	tests := []struct {
		name       string
		method     string
		body       string
		headers    map[string]string
		wantStatus int
	}{
		{
			name:       "wrong method",
			method:     http.MethodGet,
			body:       `{"login":"u","password":"p"}`,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "invalid json",
			method: http.MethodPost,
			body:   `{`,
		},
		{
			name:   "empty login",
			method: http.MethodPost,
			body:   `{"login":"","password":"p"}`,
		},
		{
			name:   "empty password",
			method: http.MethodPost,
			body:   `{"login":"u","password":""}`,
		},
		{
			name:   "invalid content type",
			method: http.MethodPost,
			body:   `{"login":"u","password":"p"}`,
			headers: map[string]string{
				"Content-Type": "text/plain",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantStatus := tc.wantStatus
			if wantStatus == 0 {
				wantStatus = http.StatusBadRequest
			}

			req := httptest.NewRequest(tc.method, "/api/user/register", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}

			rec := httptest.NewRecorder()
			h.Register(rec, req)

			if rec.Code != wantStatus {
				t.Fatalf("status: got %d, want %d", rec.Code, wantStatus)
			}
		})
	}
}

func TestRegisterStorageError(t *testing.T) {
	storage := &mockUserStorage{
		createUser: func(ctx context.Context, login, passwordHash string) (int64, error) {
			return 0, errors.New("db down")
		},
	}

	h := NewUserHandler(storage)
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(`{"login":"user","password":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d", rec.Code)
	}
}
