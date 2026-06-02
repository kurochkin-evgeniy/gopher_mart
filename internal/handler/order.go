// Package handler содержит HTTP-хендлеры бизнес-логики.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/kurochkin-evgeniy/gopher_mart/internal/logging"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/middleware"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/storage/postgres"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/validate"
)

// OrderStorage описывает операции хранилища заказов.
type OrderStorage interface {
	GetOrderByNumber(ctx context.Context, number string) (postgres.Order, int64, error)
	CreateOrder(ctx context.Context, userID int64, number string) error
	ListOrdersByUser(ctx context.Context, userID int64) ([]postgres.Order, error)
}

// OrderHandler реализует HTTP-хендлеры для работы с заказами.
type OrderHandler struct {
	storage OrderStorage
}

// NewOrderHandler создаёт хендлер заказов.
func NewOrderHandler(storage OrderStorage) *OrderHandler {
	return &OrderHandler{storage: storage}
}

type orderResponse struct {
	Number     string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    *float64  `json:"accrual,omitempty"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// Upload обрабатывает загрузку номера заказа: POST /api/user/orders.
func (h *OrderHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	number, ok := readOrderNumber(w, r)
	if !ok {
		return
	}

	order, ownerID, err := h.storage.GetOrderByNumber(r.Context(), number)
	if err == nil {
		if ownerID == userID {
			w.WriteHeader(http.StatusOK)
			return
		}
		_ = order
		http.Error(w, "order already uploaded by another user", http.StatusConflict)
		return
	}
	if !errors.Is(err, postgres.ErrOrderNotFound) {
		logging.Sugar.Errorw("get order by number", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := h.storage.CreateOrder(r.Context(), userID, number); err != nil {
		if errors.Is(err, postgres.ErrOrderTaken) {
			order, ownerID, lookupErr := h.storage.GetOrderByNumber(r.Context(), number)
			if lookupErr != nil {
				logging.Sugar.Errorw("get order after conflict", "error", lookupErr)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			if ownerID == userID {
				w.WriteHeader(http.StatusOK)
				return
			}
			_ = order
			http.Error(w, "order already uploaded by another user", http.StatusConflict)
			return
		}
		logging.Sugar.Errorw("create order", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// List возвращает список заказов пользователя: GET /api/user/orders.
func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	orders, err := h.storage.ListOrdersByUser(r.Context(), userID)
	if err != nil {
		logging.Sugar.Errorw("list orders", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	response := make([]orderResponse, 0, len(orders))
	for _, order := range orders {
		response = append(response, orderResponse{
			Number:     order.Number,
			Status:     order.Status,
			Accrual:    order.Accrual,
			UploadedAt: order.UploadedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logging.Sugar.Errorw("encode orders response", "error", err)
	}
}

func readOrderNumber(w http.ResponseWriter, r *http.Request) (string, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logging.Sugar.Warnw("read order body", "error", err)
		http.Error(w, "invalid request format", http.StatusBadRequest)
		return "", false
	}

	number := strings.TrimSpace(string(body))
	if number == "" {
		http.Error(w, "invalid request format", http.StatusBadRequest)
		return "", false
	}

	for _, r := range number {
		if !unicode.IsDigit(r) {
			http.Error(w, "invalid order number format", http.StatusUnprocessableEntity)
			return "", false
		}
	}

	if !validate.OrderNumber(number) {
		http.Error(w, "invalid order number format", http.StatusUnprocessableEntity)
		return "", false
	}

	return number, true
}
