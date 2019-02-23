package handlers

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	goji "goji.io"
	"goji.io/pat"
)

// RegisterMetricsHandler registers our metrics handler created by Prometheus at
// the expected path
func RegisterMetricsHandler(mux *goji.Mux) {
	mux.Handle(pat.Get("/metrics"), promhttp.Handler())
}
