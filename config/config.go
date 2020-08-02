package config

import (
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// Logger configuration structure.
type Logger struct {
	Level           string
	DevelopmentMode bool
}

// Server configuration structure.
type Server struct {
	Port           int
	ShutdownPeriod time.Duration
}

// Configuration structure.
type Configuration struct {
	Logging Logger
	Server  Server
}

// Parse and validate viper config.
func Parse() (cfg Configuration, err error) {
	if err := viper.ReadInConfig(); err != nil {
		return cfg, errors.Wrap(err, "failed to read configuration")
	}

	setDefaults()

	if err := viper.Unmarshal(&cfg); err != nil {
		return cfg, errors.Wrap(err, "failed to deserialize config")
	}

	return cfg, nil
}

func setDefaults() {
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("server.port", 8079)
}
