package utils

import (
	"go.uber.org/zap"
)

var Logger *zap.Logger

func InitLogger(mode string) error {
	var err error
	if mode == "production" {
		Logger, err = zap.NewProduction()
	} else {
		Logger, err = zap.NewDevelopment()
	}

	if err != nil {
		return err
	}

	return nil
}

func GetLogger() *zap.Logger {
	if Logger == nil {
		Logger, _ = zap.NewDevelopment()
	}
	return Logger
}
