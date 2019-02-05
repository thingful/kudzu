package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/thingful/kuzu/pkg/http/middleware"
	"github.com/thingful/kuzu/pkg/postgres"
)

// Error is an interface for an error type we return from our custom handler
// type.
type Error interface {
	error
	Status() int
}

// StatusError is our concrete implementation of the Error interface we return
// from handlers
type StatusError struct {
	Code int    `json:"Name"`
	Msg  string `json:"Message"`
}

// Error returns the message
func (se *StatusError) Error() string {
	return se.Msg
}

// Status returns the status code associated with the error response.
func (se *StatusError) Status() int {
	return se.Code
}

// Env is used to pass in our database and indexer to handlers
type Env struct {
	db *postgres.DB
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
				logger := middleware.LoggerFromContext(r.Context())
				logger.Log("msg", "internal server error", "error", e.Error())
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
			logger := middleware.LoggerFromContext(r.Context())
			logger.Log("msg", "internal server error", "error", err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}