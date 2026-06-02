// Package main запускает HTTP-сервис накопительной системы "Гофермарт".
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kurochkin-evgeniy/gopher_mart/internal/accrual"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/auth"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/config"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/handler"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/logging"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/router"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/storage/postgres"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/worker"
)

func main() {
	if err := logging.Init(); err != nil {
		panic(err)
	}
	defer logging.Sync()

	cfg := config.Load()
	auth.Init(cfg.JWTSecret)

	if cfg.DatabaseURI == "" {
		logging.Sugar.Fatal("DATABASE_URI is required")
	}

	appCtx, stopApp := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopApp()

	storage, err := postgres.New(context.Background(), cfg.DatabaseURI)
	if err != nil {
		logging.Sugar.Fatalw("init storage", "error", err)
	}
	defer storage.Close()

	userHandler := handler.NewUserHandler(storage)
	orderHandler := handler.NewOrderHandler(storage)
	balanceHandler := handler.NewBalanceHandler(storage)
	srv := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: router.New(userHandler, orderHandler, balanceHandler),
	}

	var workerWG sync.WaitGroup
	if cfg.AccrualSystemAddress != "" {
		accrualClient := accrual.NewClient(cfg.AccrualSystemAddress)
		accrualWorker := worker.NewAccrualWorker(storage, accrualClient, time.Second)
		workerWG.Add(1)
		go func() {
			defer workerWG.Done()
			accrualWorker.Run(appCtx)
		}()
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

	<-appCtx.Done()
	logging.Sugar.Infow("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logging.Sugar.Errorw("server shutdown", "error", err)
	}

	workerWG.Wait()
}
