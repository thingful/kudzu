package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/thingful/kuzu/pkg/client"
	"github.com/thingful/kuzu/pkg/flowerpower"
	"github.com/thingful/kuzu/pkg/indexer"
	"github.com/thingful/kuzu/pkg/logger"
	"github.com/thingful/kuzu/pkg/postgres"
	"github.com/thingful/kuzu/pkg/thingful"
)

// Thingful is the interface we expect for a type of object that can speak to
// the Thingful backend to write or read data.
type Thingful interface {
	// CreateThing attempts to create a new entry in the Thingful data store,
	// returning a generated UID or an error
	CreateThing(context.Context, *postgres.Thing, []flowerpower.Reading) (string, error)

	// UpdateThing attempts to update an existing Thing, specifically its title,
	// indexed_at, updated_at, and any channels. Used for writing time series data.
	UpdateThing(context.Context, *postgres.Thing, []flowerpower.Reading) error

	// GetData attempts to return a slice of objects read from the Thingful core
	// API. This is how this component returns time series data to any caller.
	GetData(context.Context, []string, time.Time, time.Time, bool) ([]thingful.Thing, error)
}

// Env is used to pass in our database and indexer environment to handlers
type Env struct {
	db       *postgres.DB
	client   *client.Client
	indexer  *indexer.Indexer
	thingful Thingful
}

// Handler is a custom handler type that provides some error handling niceties.
type Handler struct {
	env     *Env
	handler func(env *Env, w http.ResponseWriter, r *http.Request) error
}

// ServeHTTP is our implementation of the Handler interface
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h.handler(h.env, w, r)
	if err != nil {
		switch e := err.(type) {
		case Error:
			// log some extra stuff if this is a non-client error
			if e.Status() == http.StatusInternalServerError {
				log := logger.FromContext(r.Context())
				log.Log("msg", "internal server error", "error", e.Error())
			}

			// now marshal to JSON
			b, innerErr := json.Marshal(e)
			if innerErr != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(e.Status())
			w.Write(b)
		default:
			log := logger.FromContext(r.Context())
			log.Log("msg", "internal server error", "error", err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}
