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
			Help:      "A count of users partitioned by auth provider",
		}, []string{"provider"},
	)

	thingsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "grow",
			Name:      "things",
			Help:      "A count of things partitioned by provider and status",
		}, []string{"provider", "status"},
	)

	identitiesGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "grow",
			Name:      "identities",
			Help:      "A count of identities partitioned by status",
		}, []string{"status"},
	)
)

func init() {
	prometheus.MustRegister(usersGauge)
	prometheus.MustRegister(thingsGauge)
	prometheus.MustRegister(identitiesGauge)
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

	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(60))

		ctx := logger.ToContext(context.Background(), a.logger)

		for range ticker.C {
			userStats, err := a.db.CountUsers(ctx)
			if err != nil {
				a.logger.Log(
					"msg", "failed to read user stats",
					"error", err,
				)
				continue
			}

			for _, userStat := range userStats {
				usersGauge.With(
					prometheus.Labels{
						"provider": userStat.Provider,
					},
				).Set(userStat.Count)
			}

			identityStat, err := a.db.GetIdentityStats(ctx)
			if err != nil {
				a.logger.Log(
					"msg", "failed to read identity stats",
					"error", err,
				)
				continue
			}

			identitiesGauge.With(
				prometheus.Labels{
					"status": "all",
				},
			).Set(identityStat.All)

			identitiesGauge.With(
				prometheus.Labels{
					"status": "pending",
				},
			).Set(identityStat.Pending)

			identitiesGauge.With(
				prometheus.Labels{
					"status": "stale",
				},
			).Set(identityStat.Stale)

			thingStats, err := a.db.GetThingStats(ctx)
			if err != nil {
				a.logger.Log(
					"msg", "failed to read thing stats",
					"error", err,
				)
				continue
			}

			for _, thingStat := range thingStats {
				thingsGauge.With(
					prometheus.Labels{
						"provider": thingStat.Provider,
						"status":   "all",
					},
				).Set(thingStat.All)

				thingsGauge.With(
					prometheus.Labels{
						"provider": thingStat.Provider,
						"status":   "stale",
					},
				).Set(thingStat.Stale)

				thingsGauge.With(
					prometheus.Labels{
						"provider": thingStat.Provider,
						"status":   "invalid_location",
					},
				).Set(thingStat.InvalidLocation)
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
