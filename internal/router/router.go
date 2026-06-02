// Package router формирует маршрутизацию HTTP API.
package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/handler"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/logging"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/middleware"
)

// New настраивает маршрутизатор с зарегистрированными обработчиками API.
func New(
	userHandler *handler.UserHandler,
	orderHandler *handler.OrderHandler,
	balanceHandler *handler.BalanceHandler,
) http.Handler {
	r := chi.NewRouter()

	r.Use(requestLogger)

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", userHandler.Register)
		r.Post("/login", userHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate)
			r.Post("/orders", orderHandler.Upload)
			r.Get("/orders", orderHandler.List)
			r.Get("/balance", balanceHandler.GetBalance)
			r.Post("/balance/withdraw", balanceHandler.Withdraw)
			r.Get("/withdrawals", balanceHandler.ListWithdrawals)
		})
	})

	return r
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		logging.Sugar.Infow("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration", time.Since(start),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}
