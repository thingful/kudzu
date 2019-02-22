package client

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// InstrumentRoundTripperDuration is a helper function that copies the
// implementation provided as part of the promhttp package, but we also
// partition by requested host as we know that we only ever request from two
// hosts.
func InstrumentRoundTripperDuration(obs prometheus.ObserverVec, next http.RoundTripper) promhttp.RoundTripperFunc {
	return promhttp.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		start := time.Now()
		resp, err := next.RoundTrip(r)
		if err == nil {
			obs.With(
				prometheus.Labels{
					"code":   resp.Status,
					"method": r.Method,
					"host":   r.URL.Host,
				},
			).Observe(time.Since(start).Seconds())
		}
		return resp, err
	})
}
