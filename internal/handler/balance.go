// Package handler содержит HTTP-хендлеры бизнес-логики.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/kurochkin-evgeniy/gopher_mart/internal/logging"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/middleware"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/model"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/validate"
)

// BalanceStorage описывает операции хранилища для баланса и списаний.
type BalanceStorage interface {
	GetBalance(ctx context.Context, userID int64) (model.Balance, error)
	Withdraw(ctx context.Context, userID int64, orderNumber string, sum float64) error
	ListWithdrawals(ctx context.Context, userID int64) ([]model.Withdrawal, error)
}

// BalanceHandler реализует HTTP-хендлеры баланса и списаний.
type BalanceHandler struct {
	storage BalanceStorage
}

// NewBalanceHandler создаёт хендлер баланса.
func NewBalanceHandler(storage BalanceStorage) *BalanceHandler {
	return &BalanceHandler{storage: storage}
}

type balanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type withdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

type withdrawalResponse struct {
	Order       string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

// GetBalance возвращает текущий баланс пользователя: GET /api/user/balance.
func (h *BalanceHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	balance, err := h.storage.GetBalance(r.Context(), userID)
	if err != nil {
		logging.Sugar.Errorw("get balance", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(balanceResponse{
		Current:   balance.Current,
		Withdrawn: balance.Withdrawn,
	}); err != nil {
		logging.Sugar.Errorw("encode balance response", "error", err)
	}
}

// Withdraw регистрирует списание баллов: POST /api/user/balance/withdraw.
func (h *BalanceHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req withdrawRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		logging.Sugar.Warnw("invalid withdraw request", "error", err)
		http.Error(w, "invalid request format", http.StatusBadRequest)
		return
	}

	if req.Order == "" || req.Sum <= 0 {
		http.Error(w, "invalid request format", http.StatusBadRequest)
		return
	}

	if !validate.OrderNumber(req.Order) {
		http.Error(w, "invalid order number format", http.StatusUnprocessableEntity)
		return
	}

	if err := h.storage.Withdraw(r.Context(), userID, req.Order, req.Sum); err != nil {
		if errors.Is(err, model.ErrInsufficientFunds) {
			http.Error(w, "insufficient funds", http.StatusPaymentRequired)
			return
		}
		logging.Sugar.Errorw("withdraw", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ListWithdrawals возвращает историю списаний: GET /api/user/withdrawals.
func (h *BalanceHandler) ListWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	withdrawals, err := h.storage.ListWithdrawals(r.Context(), userID)
	if err != nil {
		logging.Sugar.Errorw("list withdrawals", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	response := make([]withdrawalResponse, 0, len(withdrawals))
	for _, item := range withdrawals {
		response = append(response, withdrawalResponse{
			Order:       item.Order,
			Sum:         item.Sum,
			ProcessedAt: item.ProcessedAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logging.Sugar.Errorw("encode withdrawals response", "error", err)
	}
}
