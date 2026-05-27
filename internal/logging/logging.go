package logging

import "go.uber.org/zap"

var Sugar *zap.SugaredLogger

func Init() error {
	logger, err := zap.NewProduction()
	if err != nil {
		return err
	}
	Sugar = logger.Sugar()
	return nil
}

func Sync() {
	if Sugar != nil {
		_ = Sugar.Sync()
	}
}
