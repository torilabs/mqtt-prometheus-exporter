package server

import (
	"net/http"
	"time"

	"github.com/etherlabsio/healthcheck"
)

// RegisterHealthCheckHandler registers healthcheck endpoint and binds it to healthcheck handler composed of provided healthcheck.Option array.
func RegisterHealthCheckHandler(options []healthcheck.Option) {
	options = append(options, healthcheck.WithTimeout(5*time.Second))

	handler := healthcheck.Handler(options...)

	http.Handle("/healthcheck", handler)
}
