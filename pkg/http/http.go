package http

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/thingful/kuzu/pkg/http/handlers"
	"github.com/thingful/kuzu/pkg/postgres"
	goji "goji.io"

	kitlog "github.com/go-kit/kit/log"
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
		"msg", "configuring http service",
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
	h.logger.Log("msg", "starting http service")

	mux := goji.NewMux()
	handlers.RegisterHealthCheck(mux, h.DB)

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
