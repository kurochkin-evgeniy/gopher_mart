// Package worker реализует фоновую обработку заказов через систему начислений.
package worker

import (
	"context"
	"errors"
	"time"

	"github.com/kurochkin-evgeniy/gopher_mart/internal/accrual"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/logging"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/storage/postgres"
)

// OrderStorage описывает операции хранилища, необходимые воркеру.
type OrderStorage interface {
	ListOrdersForAccrual(ctx context.Context) ([]string, error)
	UpdateOrderAccrual(ctx context.Context, number, status string, accrual *float64) error
}

// AccrualClient описывает клиент системы начислений.
type AccrualClient interface {
	GetOrder(ctx context.Context, number string) (*accrual.OrderInfo, error)
}

// AccrualWorker периодически опрашивает систему начислений для незавершённых заказов.
type AccrualWorker struct {
	storage OrderStorage
	client  AccrualClient
	interval time.Duration
}

// NewAccrualWorker создаёт воркер опроса системы начислений.
func NewAccrualWorker(storage OrderStorage, client AccrualClient, interval time.Duration) *AccrualWorker {
	if interval <= 0 {
		interval = time.Second
	}
	return &AccrualWorker{
		storage:  storage,
		client:   client,
		interval: interval,
	}
}

// Run запускает цикл обработки заказов до отмены контекста.
func (w *AccrualWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processOrders(ctx)
		}
	}
}

// ProcessOnce выполняет одну итерацию обработки (удобно для тестов).
func (w *AccrualWorker) ProcessOnce(ctx context.Context) {
	w.processOrders(ctx)
}

func (w *AccrualWorker) processOrders(ctx context.Context) {
	numbers, err := w.storage.ListOrdersForAccrual(ctx)
	if err != nil {
		logging.Sugar.Errorw("list orders for accrual", "error", err)
		return
	}

	for _, number := range numbers {
		if ctx.Err() != nil {
			return
		}

		info, err := w.client.GetOrder(ctx, number)
		if err != nil {
			var tooMany accrual.ErrTooManyRequests
			if errors.As(err, &tooMany) {
				logging.Sugar.Warnw("accrual rate limit", "order", number, "retry_after", tooMany.RetryAfter)
				time.Sleep(tooMany.RetryAfter)
				continue
			}
			if errors.Is(err, accrual.ErrNotRegistered) {
				continue
			}
			logging.Sugar.Errorw("get order from accrual", "order", number, "error", err)
			continue
		}

		status := accrual.MapStatus(info.Status)
		if err := w.storage.UpdateOrderAccrual(ctx, number, status, info.Accrual); err != nil {
			logging.Sugar.Errorw("update order accrual", "order", number, "error", err)
		}
	}
}

// Ensure postgres.Storage implements OrderStorage.
var _ OrderStorage = (*postgres.Storage)(nil)
