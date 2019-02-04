package handlers

import (
	"github.com/heptiolabs/healthcheck"
	goji "goji.io"
	"goji.io/pat"
)

func RegisterHealthCheck(mux *goji.Mux) {
	health := healthcheck.NewHandler()

	health.AddLivenessCheck("goroutine-threshold", healthcheck.GoroutineCountCheck(100))

	mux.HandleFunc(pat.Get("/ready"), health.ReadyEndpoint)
	mux.HandleFunc(pat.Get("/live"), health.LiveEndpoint)
}
