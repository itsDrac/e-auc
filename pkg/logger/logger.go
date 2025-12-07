package logger

import (
	"github.com/itsDrac/e-auc/pkg/utils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.SugaredLogger
}

// NewLogger initialize a new Zap Logger
// based on environment it switch to console or json
func NewLogger() *Logger {
	env := utils.GetEnv("GO_ENV", "development")

	var cfg zap.Config

	if env == "production" {
		cfg = zap.NewProductionConfig()
		cfg.Encoding = "json"
		cfg.EncoderConfig.TimeKey = "ts"
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	log, err := cfg.Build()
	if err != nil {
		panic("[LOGGER] failed to initialize ->" + err.Error())
	}

	return &Logger{
		log.Sugar(),
	}
}
