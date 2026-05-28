// Package main запускает HTTP-сервис накопительной системы "Гофермарт".
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kurochkin-evgeniy/gopher_mart/internal/config"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/handler"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/logging"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/router"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/storage/postgres"
)

func main() {
	if err := logging.Init(); err != nil {
		panic(err)
	}
	defer logging.Sync()

	cfg := config.Load()

	if cfg.DatabaseURI == "" {
		logging.Sugar.Fatal("DATABASE_URI is required")
	}

	ctx := context.Background()
	storage, err := postgres.New(ctx, cfg.DatabaseURI)
	if err != nil {
		logging.Sugar.Fatalw("init storage", "error", err)
	}
	defer storage.Close()

	userHandler := handler.NewUserHandler(storage)
	srv := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: router.New(userHandler),
	}

	go func() {
		logging.Sugar.Infow("starting server",
			"address", cfg.RunAddress,
			"accrual_system", cfg.AccrualSystemAddress,
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logging.Sugar.Fatalw("listen and serve", "error", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logging.Sugar.Errorw("server shutdown", "error", err)
	}
}
