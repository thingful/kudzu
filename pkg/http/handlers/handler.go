package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/thingful/kuzu/pkg/client"
	"github.com/thingful/kuzu/pkg/logger"
	"github.com/thingful/kuzu/pkg/postgres"
)

// Error is an interface for an error type we return from our custom handler
// type.
type Error interface {
	error
	Status() int
}

// HTTPError is our concrete implementation of the Error interface we return
// from handlers
type HTTPError struct {
	Code int
	Err  error
}

// Error returns the message
func (he *HTTPError) Error() string {
	return he.Err.Error()
}

// Status returns the status code associated with the error response.
func (he *HTTPError) Status() int {
	return he.Code
}

// MarshalJSON is our implementation of the json marshaller interface as we want
// to output the error message rather than the default error serialization which
// seems to be an empty json object.
func (he *HTTPError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Code    int    `json:"Name"`
		Message string `json:"Message"`
	}{
		Code:    he.Code,
		Message: he.Err.Error(),
	})
}

// Env is used to pass in our database and indexer environment to handlers
type Env struct {
	db     *postgres.DB
	client *client.Client
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
