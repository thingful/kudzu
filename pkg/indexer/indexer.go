package indexer

import (
	"context"
	"sync"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/google/uuid"

	"github.com/thingful/kuzu/pkg/client"
	"github.com/thingful/kuzu/pkg/logger"
	"github.com/thingful/kuzu/pkg/postgres"
	"github.com/thingful/kuzu/pkg/thingful"
)

// Config is another state holder we pass in to the indexer to configure it.
type Config struct {
	DB        *postgres.DB
	Client    *client.Client
	QuitChan  <-chan struct{}
	ErrChan   chan<- error
	WaitGroup *sync.WaitGroup
	Delay     time.Duration
}

// Indexer is a struct that controls the scheduled work where we pull data from
// Parrot and write it to Thingful.
type Indexer struct {
	*Config
	Thingful *thingful.Thingful
	logger   kitlog.Logger
}

// NewIndexer returns a new Indexer instance ready to start work.
func NewIndexer(config *Config, logger kitlog.Logger) *Indexer {
	logger = kitlog.With(logger, "module", "indexer")

	logger.Log("msg", "configuring indexer", "delay", config.Delay)

	return &Indexer{
		Config: config,
		logger: logger,
	}
}

// Start starts our indexer running, any errors sent back via the error channel
func (i *Indexer) Start() {
	i.logger.Log("msg", "starting indexer")

	ticker := time.NewTicker(i.Delay)

	for {
		select {
		case <-ticker.C:
			i.Index()
		case <-i.QuitChan:
			i.logger.Log("msg", "stopping indexer")
			ticker.Stop()
			i.WaitGroup.Done()
			return
		}
	}
}

// Index is called repeatedly - attempts to get an identity for indexing, and we
// then index all of that user's unindexed stuff.
func (i *Indexer) Index() {
	uid := uuid.New().String()

	log := kitlog.With(i.logger, "uid", uid)

	accessToken, err := i.DB.NextAccessToken(logger.ToContext(context.Background(), log))
	if err != nil {
		log.Log("msg", "error getting next identiyt", "err", err)
	}

	if accessToken == "" {
		return
	}

	err = i.IndexLocations(accessToken)
	if err != nil {
		i.logger.Log("msg", "error indexing locations", "err", err)
	}
}

// IndexLocations is the entry point to our fetching and parsing logic - indexes
// all unindexed data for a user and publishes to Thingful
func (i *Indexer) IndexLocations(accessToken string) error {
	i.logger.Log("msg", "indexing locations", "accessToken", accessToken)

	return nil
}

// Index is the function called recursively to do some task, we start it off when
// the process starts, once it has finished doing a task, it sleeps and then
// calls itself again.
//func (i *Indexer) Index() {
//	i.logger.Log("msg", "indexing a resource")
//	ctx := logger.ToContext(context.Background(), i.logger)
//
//	// get a location that has never been published to Thingful, then send a create request to create it
//	newLocation, err := i.DB.GetNewLocation(ctx)
//	if err != nil {
//		i.logger.Log("msg", "error getting new location", "err", err)
//		return
//	}
//
//	// we have a location that has never been published to Thingful, so we need to
//	// create a new Thingful resource for this location
//	if newLocation != nil {
//		thing, err := i.Thingful.CreateThing(ctx, newLocation)
//		if err != nil {
//			i.logger.Log("msg", "error publishing new thing to Thingful", "err", err)
//			return
//		}
//
//		err = i.DB.SetLocationUID(ctx, thing.UID)
//		if err != nil {
//			i.logger.Log("msg", "error updating new location", "err", err)
//		}
//
//		return
//	}
//
//	existingLocation, err := i.DB.GetIndexableLocation(ctx)
//	if err != nil {
//		i.logger.Log("msg", "error getting indexable location", "err", err)
//		return
//	}
//
//	if existingLocation != nil {
//		err = i.indexLocation(ctx, existingLocation)
//		if err != nil {
//			i.logger.Log("msg", "error indexing location", "err", err)
//		}
//	}
//
//	return
//}
//
//func (i *Indexer) indexLocation(ctx context.Context, location *postgres.Location) error {
//	return nil
//}
