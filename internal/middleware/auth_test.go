package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kurochkin-evgeniy/gopher_mart/internal/auth"
)

func TestAuthenticate(t *testing.T) {
	token, err := auth.GenerateToken(99)
	if err != nil {
		t.Fatal(err)
	}

	var gotUserID int64
	handler := Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := UserIDFromContext(r.Context())
		if !ok {
			t.Fatal("user id not in context")
		}
		gotUserID = id
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	req.Header.Set(auth.AuthHeader, auth.AuthScheme+" "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d", rec.Code)
	}
	if gotUserID != 99 {
		t.Fatalf("user id: got %d", gotUserID)
	}
}

func TestAuthenticateUnauthorized(t *testing.T) {
	handler := Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d", rec.Code)
	}
}
