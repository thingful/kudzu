package handlers

import (
	"time"

	"github.com/heptiolabs/healthcheck"
	goji "goji.io"
	"goji.io/pat"

	"github.com/thingful/kudzu/pkg/postgres"
)

// RegisterHealthCheck registers a couple of endpoints using the heptiolabs
// healthcheck library. These add a /live which reports if the service is or
// should be restarted, and /ready which reports if the service is up and ready
// to handle work. We typically don't restart if this returns an error response
// as this indicates an upstream failure that we should wait for.
func RegisterHealthCheck(mux *goji.Mux, db *postgres.DB) {
	health := healthcheck.NewHandler()

	health.AddLivenessCheck("goroutine-threshold", healthcheck.GoroutineCountCheck(100))
	health.AddReadinessCheck("postgres", healthcheck.DatabasePingCheck(db.DB.DB, 1*time.Second))

	mux.HandleFunc(pat.Get("/ready"), health.ReadyEndpoint)
	mux.HandleFunc(pat.Get("/live"), health.LiveEndpoint)
}
