// Package logging предоставляет инициализацию логгирования для всего приложения.
package logging

import "go.uber.org/zap"

// Sugar является глобальным SugaredLogger для логирования из разных пакетов.
var Sugar *zap.SugaredLogger

// Init инициализирует Sugar (zap production logger).
func Init() error {
	logger, err := zap.NewProduction()
	if err != nil {
		return err
	}
	Sugar = logger.Sugar()
	return nil
}

// Sync принудительно сбрасывает буфер логгера (если он инициализирован).
func Sync() {
	if Sugar != nil {
		_ = Sugar.Sync()
	}
}
