package app

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/thingful/kuzu/pkg/client"
	"github.com/thingful/kuzu/pkg/http"
	"github.com/thingful/kuzu/pkg/indexer"
	"github.com/thingful/kuzu/pkg/logger"
	"github.com/thingful/kuzu/pkg/postgres"
	"github.com/thingful/kuzu/pkg/thingful"
)

var (
	usersGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "grow",
			Name:      "registered_users",
			Help:      "Count of users partitioned by auth provider",
		}, []string{"provider"},
	)
)

func init() {
	prometheus.MustRegister(usersGauge)
}

// Config is our top level config struct used to carry all configuration from
// cobra commands into our application code.
type Config struct {
	Addr          string
	DatabaseURL   string
	ClientTimeout int
	Verbose       bool
	Delay         int
	ThingfulURL   string
	ThingfulKey   string
	Concurrency   int
}

// NewApp returns a new App instance with components configured but not yet
// started.
func NewApp(config *Config) *App {
	logger := logger.NewLogger()

	db := postgres.NewDB(config.DatabaseURL, config.Verbose)

	cl := client.NewClient(config.ClientTimeout, config.Verbose)

	th := thingful.NewClient(cl, config.ThingfulURL, config.ThingfulKey, config.Verbose, config.Concurrency)

	quitChan := make(chan struct{})
	errChan := make(chan error)
	var wg sync.WaitGroup

	i := indexer.NewIndexer(&indexer.Config{
		DB:        db,
		Client:    cl,
		QuitChan:  quitChan,
		ErrChan:   errChan,
		WaitGroup: &wg,
		Delay:     time.Duration(config.Delay) * time.Second,
		Thingful:  th,
		Verbose:   config.Verbose,
	}, logger)

	h := http.NewHTTP(&http.Config{
		DB:        db,
		Client:    cl,
		Thingful:  th,
		Addr:      config.Addr,
		QuitChan:  quitChan,
		ErrChan:   errChan,
		WaitGroup: &wg,
		Indexer:   i,
	}, logger)

	return &App{
		logger:  kitlog.With(logger, "module", "app"),
		http:    h,
		db:      db,
		indexer: i,

		quitChan: quitChan,
		errChan:  errChan,
		wg:       &wg,
	}
}

// App is our core application instance - holds all the state and child
// components and is responsible for starting/stopping and managing
// communication between these elements.
type App struct {
	logger  kitlog.Logger
	db      *postgres.DB
	http    *http.HTTP
	indexer *indexer.Indexer

	quitChan chan struct{}
	errChan  chan error
	wg       *sync.WaitGroup
}

// Start the application running. We spawn some child components in separate
// goroutines and hook up some channels by which we can communicate with these
// tasks.
func (a *App) Start() error {
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

	go func() {
		a.wg.Add(1)
		a.indexer.Start()
	}()

	ticker := time.NewTicker(time.Second * time.Duration(30))

	go func() {
		for range ticker.C {
			userStats, err := a.db.CountUsers(context.Background())
			if err != nil {
				a.logger.Log(
					"msg", "failed to read user stats",
					"error", err,
				)
				continue
			}

			for _, stat := range userStats {
				usersGauge.With(
					prometheus.Labels{
						"provider": stat.Provider,
					},
				).Set(stat.Count)
			}
		}
	}()

	a.logger.Log("msg", "starting app")

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
