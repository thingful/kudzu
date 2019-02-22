package middleware

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	duration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "grow",
			Name:      "server_request_duration_seconds",
			Help:      "A histogram of the latency in seconds for serving requests",
			Buckets:   prometheus.DefBuckets,
		}, []string{"code", "method"},
	)
)

func init() {
	prometheus.MustRegister(duration)
}

// MetricsMiddleware returns a handler that records some basic prometheus
// metrics on the wrapped Handler. Currently we just record the duration of
// requests to get a measure of the latency of responses, partitioned by method
// and status code.
func MetricsMiddleware(next http.Handler) http.Handler {
	return promhttp.InstrumentHandlerDuration(duration, next)
}
