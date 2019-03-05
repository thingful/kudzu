package indexer

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/guregu/null"
	"github.com/prometheus/client_golang/prometheus"

	kitlog "github.com/go-kit/kit/log"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/thingful/kudzu/pkg/client"
	"github.com/thingful/kudzu/pkg/flowerpower"
	"github.com/thingful/kudzu/pkg/logger"
	"github.com/thingful/kudzu/pkg/postgres"
	"github.com/thingful/kudzu/pkg/thingful"
)

var (
	indexCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "grow",
			Name:      "indexed_identity",
			Help:      "A counter that increments every time we index a new identity",
		},
	)
)

// Config is another state holder we pass in to the indexer to configure it.
type Config struct {
	DB        *postgres.DB
	Client    *client.Client
	Thingful  *thingful.Thingful
	QuitChan  <-chan struct{}
	ErrChan   chan<- error
	WaitGroup *sync.WaitGroup
	Delay     time.Duration
	Verbose   bool
	NoIndexer bool
}

// Indexer is a struct that controls the scheduled work where we pull data from
// Parrot and write it to Thingful.
type Indexer struct {
	*Config
	logger kitlog.Logger
}

// NewIndexer returns a new Indexer instance ready to start work.
func NewIndexer(config *Config, logger kitlog.Logger) *Indexer {
	logger = kitlog.With(logger, "module", "indexer")

	return &Indexer{
		Config: config,
		logger: logger,
	}
}

// Start starts our indexer running, any errors sent back via the error channel
func (i *Indexer) Start() {
	if !i.NoIndexer {
		i.logger.Log("msg", "starting indexer")
	}

	ticker := time.NewTicker(i.Delay)

	for {
		select {
		case <-ticker.C:
			if !i.NoIndexer {
				i.Index()
			}
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
	// create index event uuid, and wrap this id into a logger we pass down via
	// context
	uid := uuid.New().String()
	log := kitlog.With(i.logger, "uid", uid)
	ctx := logger.ToContext(context.Background(), log)

	// next identity to index
	identity, err := i.DB.NextIdentity(ctx)
	if err != nil {
		log.Log("msg", "error getting next identity", "err", err)
	}

	// TODO: increment indexCount

	if identity.AccessToken == "" {
		if i.Verbose {
			log.Log("msg", "no pending identity found")
		}
		return
	}

	// now index all locations for the identity
	err = i.indexLocations(ctx, identity)
	if err != nil {
		i.logger.Log("msg", "error indexing locations", "err", err)
	}
}

// indexLocations is the entry point to our fetching and parsing logic - indexes
// all unindexed data for a user and publishes to Thingful
func (i *Indexer) indexLocations(ctx context.Context, identity *postgres.Identity) error {
	// this seems cumbersome as we could pass the logger in directly here, however
	// this function also called from the user create handler, which will have a
	// differently scoped logger
	log := logger.FromContext(ctx)

	if i.Verbose {
		log.Log("msg", "indexing locations", "ownerID", identity.OwnerID)
	}

	// get the locations from parrot (makes multiple API requests)
	locations, err := flowerpower.GetLocations(ctx, i.Client, identity.AccessToken)
	if err != nil {
		log.Log("msg", "failed to get locations for indexing", "ownerID", identity.OwnerID)
		return errors.Wrap(err, "failed to get locations for indexing")
	}

	if i.Verbose {
		log.Log(
			"msg", "retrieved locations from parrot",
			"numLocations", len(locations),
		)
	}

	// now let's range over our retrieved locations
	for _, l := range locations {
		thing, err := i.DB.GetThing(ctx, l.LocationID)
		if err != nil {
			if i.Verbose {
				log.Log(
					"msg", "error getting thing from DB",
					"err", err.Error(),
				)
			}

			if errors.Cause(err) == sql.ErrNoRows {
				// launch new thing flow
				err = i.indexNewLocation(ctx, identity, &l)
				if err != nil {
					log.Log("msg", "error indexing location", "locationID", l.LocationID, "err", err)
				}
				continue
			}
			return err
		}
		// launch update thing flow
		err = i.indexExistingLocation(ctx, identity, &l, thing)
		if err != nil {
			log.Log("msg", "error indexing existing location", "locationID", l.LocationID, "err", err)
		}
	}

	return nil
}

func (i *Indexer) indexNewLocation(ctx context.Context, identity *postgres.Identity, l *flowerpower.Location) error {
	log := logger.FromContext(ctx)

	now := time.Now()

	thing := &postgres.Thing{
		OwnerID:        identity.OwnerID,
		Provider:       null.StringFrom("parrot"),
		SerialNum:      l.SerialNum,
		Longitude:      l.Longitude,
		Latitude:       l.Latitude,
		FirstSampleUTC: null.TimeFrom(l.FirstSampleUTC),
		LastSampleUTC:  null.TimeFrom(l.LastSampleUTC),
		CreatedAt:      null.TimeFrom(now),
		UpdatedAt:      null.TimeFrom(now),
		IndexedAt:      null.TimeFrom(now),
		Nickname:       null.StringFrom(l.Nickname),
		LocationID:     l.LocationID,
	}

	fromUTC := thing.FirstSampleUTC.Time
	toUTC := fromUTC.AddDate(0, 0, 10)

	if thing.LastSampleUTC.Time.Before(toUTC) {
		toUTC = thing.LastSampleUTC.Time
	}

	// get the first slice of readings for the location
	readings, err := flowerpower.GetReadings(ctx, i.Client, identity.AccessToken, l.LocationID, fromUTC, toUTC)
	if err != nil {
		return errors.Wrap(err, "failed to get first chunk of readings from flowerpower")
	}

	thingfulUID, err := i.Thingful.CreateThing(ctx, thing, readings)
	if err != nil {
		log.Log("msg", "failed to create thing", "err", err)
		return errors.Wrap(err, "failed to create new thing")
	}

	thing.UID = null.StringFrom(thingfulUID)
	thing.LastUploadedUTC = null.TimeFrom(toUTC)

	// save the thing and channels
	err = i.DB.CreateThing(ctx, thing)
	if err != nil {
		log.Log("msg", "failed to insert thing", "err", err)
		return errors.Wrap(err, "failed to insert thing record into DB")
	}

	for {
		// we sleep to avoid hammering Parrot too hard
		time.Sleep(i.Delay)

		if !hasMoreReadingsToIndex(ctx, thing) {
			break
		}

		fromUTC = thing.LastUploadedUTC.Time
		toUTC = fromUTC.AddDate(0, 0, 10)

		if thing.LastSampleUTC.Time.Before(toUTC) {
			toUTC = thing.LastSampleUTC.Time
		}

		// get next readings from flowerpower
		readings, err = flowerpower.GetReadings(ctx, i.Client, identity.AccessToken, l.LocationID, fromUTC, toUTC)
		if err != nil {
			log.Log("msg", "failed to get next readings", "err", err, "fromUTC", fromUTC, "toUTC", toUTC)
			return errors.Wrap(err, "failed to get slice of readings from Parrot")
		}

		// send to Thingful
		err = i.Thingful.UpdateThing(ctx, thing, readings)
		if err != nil {
			log.Log("msg", "failed to push observations to Thingful", "err", err, "fromUTC", fromUTC, "toUTC", toUTC)
			return errors.Wrap(err, "failed to get slice of readings from Parrot")
		}

		now = time.Now()
		thing.IndexedAt = null.TimeFrom(now)
		thing.UpdatedAt = null.TimeFrom(now)
		thing.LastUploadedUTC = null.TimeFrom(toUTC)

		// update the last uploaded timestamp and any related channels
		err = i.DB.UpdateThing(ctx, thing)
		if err != nil {
			return errors.Wrap(err, "failed to update thing")
		}
	}

	return nil
}

func (i *Indexer) indexExistingLocation(ctx context.Context, identity *postgres.Identity, location *flowerpower.Location, thing *postgres.Thing) error {
	log := logger.FromContext(ctx)

	if i.Verbose {
		log.Log(
			"msg", "indexing existing location",
			"firstSampleUTC", location.FirstSampleUTC,
			"lastSampleUTC", location.LastSampleUTC,
		)
	}

	// read the sample timestamp values from retrieved data
	thing.FirstSampleUTC = null.TimeFrom(location.FirstSampleUTC)
	thing.LastSampleUTC = null.TimeFrom(location.LastSampleUTC)

	for {
		// we sleep to avoid hammering Parrot too hard
		time.Sleep(i.Delay)

		if !hasMoreReadingsToIndex(ctx, thing) {
			break
		}

		// get the next time window to fetch
		fromUTC := thing.LastUploadedUTC.Time
		toUTC := fromUTC.AddDate(0, 0, 10)

		if i.Verbose {
			log.Log(
				"msg", "requesting chunk of data",
				"fromUTC", fromUTC,
				"toUTC", toUTC,
			)
		}

		// check if our calculated upper bound is beyond the last sample value
		if thing.LastSampleUTC.Time.Before(toUTC) {
			toUTC = thing.LastSampleUTC.Time
		}

		// get next readings from flowerpower
		readings, err := flowerpower.GetReadings(ctx, i.Client, identity.AccessToken, location.LocationID, fromUTC, toUTC)
		if err != nil {
			log.Log("msg", "failed to get next readings", "err", err, "fromUTC", fromUTC, "toUTC", toUTC)
			return errors.Wrap(err, "failed to get slice of readings from Parrot")
		}

		// send to Thingful
		err = i.Thingful.UpdateThing(ctx, thing, readings)
		if err != nil {
			log.Log("msg", "failed to push observations to Thingful", "err", err, "fromUTC", fromUTC, "toUTC", toUTC)
			return errors.Wrap(err, "failed to get slice of readings from Parrot")
		}

		now := time.Now()
		thing.IndexedAt = null.TimeFrom(now)
		thing.UpdatedAt = null.TimeFrom(now)
		thing.LastUploadedUTC = null.TimeFrom(toUTC)

		// update the last uploaded timestamp and any related channels
		err = i.DB.UpdateThing(ctx, thing)
		if err != nil {
			return errors.Wrap(err, "failed to update thing")
		}
	}

	return nil
}

// hasMoreReadingsToIndex simply checks the value of the last uploaded sample
// and compares it to the last sample sent by parrot. If the last uploaded is
// before the last value, then return true, else return false
func hasMoreReadingsToIndex(ctx context.Context, thing *postgres.Thing) bool {
	if thing.LastUploadedUTC.Time.Before(thing.LastSampleUTC.Time) {
		return true
	}
	return false
}
