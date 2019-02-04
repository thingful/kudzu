package app

import (
	"os"
	"os/signal"
	"sync"

	"github.com/thingful/kuzu/pkg/postgres"

	kitlog "github.com/go-kit/kit/log"

	"github.com/thingful/kuzu/pkg/http"
	"github.com/thingful/kuzu/pkg/logger"
)

// NewApp returns a new App instance with components configured but not yet
// started.
func NewApp(addr string, connStr string) *App {
	logger := logger.NewLogger()

	db := postgres.NewDB(connStr, logger)

	quitChan := make(chan struct{})
	errChan := make(chan error)
	var wg sync.WaitGroup

	h := http.NewHTTP(&http.Config{
		DB:        db,
		Addr:      addr,
		QuitChan:  quitChan,
		ErrChan:   errChan,
		WaitGroup: &wg,
	}, logger)

	return &App{
		logger:   kitlog.With(logger, "module", "app"),
		http:     h,
		db:       db,
		quitChan: quitChan,
		errChan:  errChan,
		wg:       &wg,
	}
}

// App is our core application instance - holds all the state and child
// components and is responsible for starting/stopping and managing
// communication between these elements.
type App struct {
	logger kitlog.Logger
	db     *postgres.DB
	http   *http.HTTP

	quitChan chan struct{}
	errChan  chan error
	wg       *sync.WaitGroup
}

// Start the application running. We spawn some child components in separate
// goroutines and hook up some channels by which we can communicate with these
// tasks.
func (a *App) Start() error {
	a.logger.Log("msg", "starting app")

	err := a.db.Start()
	if err != nil {
		return err
	}

	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)

	go func() {
		a.wg.Add(1)
		a.http.Start()
	}()

	select {
	case <-stopChan:
		a.logger.Log("msg", "stopping app")
		close(a.quitChan)
		a.wg.Wait()
	case err := <-a.errChan:
		return err
	}

	return nil
}