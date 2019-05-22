package http

import (
	"context"
	"net/http"
	"sync"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/jonboulle/clockwork"
	goji "goji.io"
	"goji.io/pat"
	"golang.org/x/time/rate"

	"github.com/thingful/kudzu/pkg/client"
	"github.com/thingful/kudzu/pkg/http/handlers"
	"github.com/thingful/kudzu/pkg/http/middleware"
	"github.com/thingful/kudzu/pkg/indexer"
	"github.com/thingful/kudzu/pkg/postgres"
	"github.com/thingful/kudzu/pkg/thingful"
)

// HTTP is our struct that exposes an HTTP server for handling incoming
// requests.
type HTTP struct {
	logger kitlog.Logger
	srv    *http.Server
	*Config
}

// Config is a struct used to pass configuration into the HTTP instance
type Config struct {
	Addr          string
	DB            *postgres.DB
	Indexer       *indexer.Indexer
	Client        *client.Client
	Thingful      *thingful.Thingful
	QuitChan      <-chan struct{}
	ErrChan       chan<- error
	WaitGroup     *sync.WaitGroup
	ServerTimeout int
	Rate          rate.Limit
	Burst         int
	Expiry        time.Duration
}

// NewHTTP returns a new HTTP instance configured and ready to use, but not yet
// started.
func NewHTTP(config *Config, logger kitlog.Logger) *HTTP {
	logger = kitlog.With(logger, "module", "http")

	srv := &http.Server{
		Addr:         config.Addr,
		ReadTimeout:  time.Duration(config.ServerTimeout) * time.Second,
		WriteTimeout: time.Duration(2*config.ServerTimeout) * time.Second,
	}

	return &HTTP{
		logger: logger,
		srv:    srv,
		Config: config,
	}
}

// Start starts the HTTP service running. Requires any dependencies to already
// be started elsewhere. Note we do return an error from the function, rather as
// we start in a separate goroutine we use a channel to report back any errors.
func (h *HTTP) Start() {
	h.logger.Log("msg", "starting http server")

	mux := goji.NewMux()
	handlers.RegisterHealthCheck(mux, h.DB)
	handlers.RegisterMetricsHandler(mux)

	apiMux := goji.SubMux()
	mux.Handle(pat.New("/api/*"), apiMux)

	handlers.RegisterUserHandlers(apiMux, h.DB, h.Client, h.Indexer)
	handlers.RegisterDataSourceHandlers(apiMux, h.DB)
	handlers.RegisterLocationHandlers(apiMux, h.DB, h.Thingful)
	handlers.RegisterMetadataHandlers(apiMux, h.DB)
	handlers.RegisterTimeseriesHandler(apiMux, h.DB, h.Thingful)
	handlers.RegisterAppHandlers(apiMux, h.DB)

	// add middleware
	apiMux.Use(middleware.RequestIDMiddleware)

	loggingMiddleware := middleware.NewLoggingMiddleware(h.logger, true)
	apiMux.Use(loggingMiddleware.Handler)

	apiMux.Use(middleware.MetricsMiddleware)

	authMiddleware := middleware.NewAuthMiddleware(h.DB)
	apiMux.Use(authMiddleware.Handler)

	rateLimitMiddleware := middleware.NewRateLimiterMiddleware(h.Config.Rate, h.Config.Burst, h.Config.Expiry, clockwork.NewRealClock())
	apiMux.Use(rateLimitMiddleware.Handler)

	h.srv.Handler = mux

	go func() {
		if err := h.srv.ListenAndServe(); err != nil {
			h.ErrChan <- err
		}
	}()

	<-h.QuitChan

	h.logger.Log("msg", "stopping http service")

	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()

	h.srv.Shutdown(ctx)
	h.WaitGroup.Done()
}
