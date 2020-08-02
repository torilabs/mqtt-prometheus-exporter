package log

import (
	"github.com/torilabs/mqtt-prometheus-exporter/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger for whole app.
var Logger *zap.SugaredLogger

func init() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	Logger = logger.Sugar()
}

// Setup initializes logger based on provided configuration.
func Setup(cfg config.Configuration) error {
	var level zapcore.Level

	if err := level.Set(cfg.Logging.Level); err != nil {
		return err
	}

	var logCfg zap.Config
	if cfg.Logging.DevelopmentMode {
		logCfg = zap.NewDevelopmentConfig()
	} else {
		logCfg = zap.NewProductionConfig()
		logCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}
	logCfg.Level.SetLevel(level)

	logger, err := logCfg.Build()
	if err != nil {
		return err
	}

	Logger = logger.Sugar()

	return nil
}
