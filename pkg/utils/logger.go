package utils

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"strings"
)

func NewLogger(level string) *zap.Logger {
	level = strings.ToUpper(level)

	var config zap.Config

	config.Encoding = "console"

	if level == "DEBUG" {
		config = zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	l, err := config.Build()
	if err != nil {
		panic("failed to instantiate logger")
	}

	return l
}
