package accrual

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetOrderSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/orders/12345678903" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"order":"12345678903","status":"PROCESSED","accrual":500}`))
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL)
	info, err := client.GetOrder(context.Background(), "12345678903")
	if err != nil {
		t.Fatal(err)
	}
	if info.Status != "PROCESSED" || info.Accrual == nil || *info.Accrual != 500 {
		t.Fatalf("unexpected info: %+v", info)
	}
}

func TestGetOrderNotRegistered(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL)
	_, err := client.GetOrder(context.Background(), "12345678903")
	if !errors.Is(err, ErrNotRegistered) {
		t.Fatalf("expected ErrNotRegistered, got %v", err)
	}
}

func TestGetOrderTooManyRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("No more than N requests per minute allowed"))
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL)
	_, err := client.GetOrder(context.Background(), "12345678903")
	var tooMany ErrTooManyRequests
	if !errors.As(err, &tooMany) {
		t.Fatalf("expected ErrTooManyRequests, got %v", err)
	}
	if tooMany.RetryAfter != 60*time.Second {
		t.Fatalf("retry after: %s", tooMany.RetryAfter)
	}
}

func TestMapStatus(t *testing.T) {
	if MapStatus("REGISTERED") != "PROCESSING" {
		t.Fatal("REGISTERED")
	}
	if MapStatus("INVALID") != "INVALID" {
		t.Fatal("INVALID")
	}
	if MapStatus("PROCESSED") != "PROCESSED" {
		t.Fatal("PROCESSED")
	}
}
