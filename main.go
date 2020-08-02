package main

import (
	"github.com/torilabs/mqtt-prometheus-exporter/cmd"
	"github.com/torilabs/mqtt-prometheus-exporter/log"
	"go.uber.org/zap"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Logger.With(zap.Error(err)).Fatal("Terminating the service.")
	}
}
