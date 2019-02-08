package indexer

import (
	"sync"
	"time"

	kitlog "github.com/go-kit/kit/log"

	"github.com/thingful/kuzu/pkg/client"
	"github.com/thingful/kuzu/pkg/postgres"
)

// Config is another state holder we pass in to the indexer to configure it.
type Config struct {
	DB        *postgres.DB
	Client    *client.Client
	QuitChan  <-chan struct{}
	ErrChan   chan<- error
	WaitGroup *sync.WaitGroup
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

	logger.Log("msg", "configuring indexer")

	return &Indexer{
		Config: config,
		logger: logger,
	}
}

// Start starts our indexer running, any errors sent back via the error channel
func (i *Indexer) Start() {
	i.logger.Log("msg", "starting indexer")

	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ticker.C:
			i.logger.Log("msg", "indexing event")
		case <-i.QuitChan:
			i.logger.Log("msg", "stopping indexer")
			ticker.Stop()
			i.WaitGroup.Done()
			return
		}
	}
}
