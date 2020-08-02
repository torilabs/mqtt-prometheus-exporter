package server

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/torilabs/mqtt-prometheus-exporter/config"
)

// ListenAndServe starts server based on provided configuration and registers request handlers.
func ListenAndServe(config config.Server) error {
	http.Handle("/metrics", promhttp.Handler())

	return http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
}
