package worker

import (
	"context"
	"testing"

	"github.com/kurochkin-evgeniy/gopher_mart/internal/accrual"
)

type mockOrderStorage struct {
	numbers  []string
	updates  []struct {
		number  string
		status  string
		accrual *float64
	}
}

func (m *mockOrderStorage) ListOrdersForAccrual(ctx context.Context) ([]string, error) {
	return m.numbers, nil
}

func (m *mockOrderStorage) UpdateOrderAccrual(ctx context.Context, number, status string, accrual *float64) error {
	m.updates = append(m.updates, struct {
		number  string
		status  string
		accrual *float64
	}{number, status, accrual})
	return nil
}

type mockAccrualClient struct {
	info *accrual.OrderInfo
	err  error
}

func (m *mockAccrualClient) GetOrder(ctx context.Context, number string) (*accrual.OrderInfo, error) {
	return m.info, m.err
}

func TestProcessOnceUpdatesOrder(t *testing.T) {
	accrualValue := 500.0
	storage := &mockOrderStorage{numbers: []string{"12345678903"}}
	client := &mockAccrualClient{
		info: &accrual.OrderInfo{
			Order:   "12345678903",
			Status:  "PROCESSED",
			Accrual: &accrualValue,
		},
	}

	w := NewAccrualWorker(storage, client, 0)
	w.ProcessOnce(context.Background())

	if len(storage.updates) != 1 {
		t.Fatalf("updates: %d", len(storage.updates))
	}
	if storage.updates[0].status != "PROCESSED" {
		t.Fatalf("status: %s", storage.updates[0].status)
	}
	if storage.updates[0].accrual == nil || *storage.updates[0].accrual != 500 {
		t.Fatal("accrual not set")
	}
}
