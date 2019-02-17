package http

import (
	"context"
	"net/http"
	"sync"
	"time"

	kitlog "github.com/go-kit/kit/log"
	goji "goji.io"

	"github.com/thingful/kuzu/pkg/client"
	"github.com/thingful/kuzu/pkg/http/handlers"
	"github.com/thingful/kuzu/pkg/http/middleware"
	"github.com/thingful/kuzu/pkg/indexer"
	"github.com/thingful/kuzu/pkg/postgres"
	"github.com/thingful/kuzu/pkg/thingful"
)

const (
	// Timeout is a timeout we add on the server to enforce timeouts for slow
	// clients
	Timeout = 5
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
	Addr      string
	DB        *postgres.DB
	Indexer   *indexer.Indexer
	Client    *client.Client
	Thingful  *thingful.Thingful
	QuitChan  <-chan struct{}
	ErrChan   chan<- error
	WaitGroup *sync.WaitGroup
}

// NewHTTP returns a new HTTP instance configured and ready to use, but not yet
// started.
func NewHTTP(config *Config, logger kitlog.Logger) *HTTP {
	logger = kitlog.With(logger, "module", "http")

	srv := &http.Server{
		Addr:         config.Addr,
		ReadTimeout:  Timeout * time.Second,
		WriteTimeout: 2 * Timeout * time.Second,
	}

	logger.Log(
		"msg", "configuring http server",
		"addr", config.Addr,
		"readTimeout", Timeout,
		"writeTimeout", 2*Timeout,
	)

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
	handlers.RegisterUserHandlers(mux, h.DB, h.Client, h.Indexer)
	handlers.RegisterDataSourceHandlers(mux, h.DB)
	handlers.RegisterLocationHandlers(mux, h.DB, h.Thingful)
	handlers.RegisterMetadataHandlers(mux, h.DB)

	// add middleware
	mux.Use(middleware.RequestIDMiddleware)

	loggingMiddleware := middleware.NewLoggingMiddleware(h.logger, true)
	mux.Use(loggingMiddleware.Handler)

	authMiddleware := middleware.NewAuthMiddleware(h.DB)
	mux.Use(authMiddleware.Handler)

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
