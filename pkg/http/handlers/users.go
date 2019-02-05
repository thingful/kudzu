package handlers

import (
	"net/http"

	goji "goji.io"
	"goji.io/pat"

	"github.com/thingful/kuzu/pkg/postgres"
)

// RegisterUserHandlers registers our user related handlers into the mux
func RegisterUserHandlers(mux *goji.Mux, db *postgres.DB) {
	mux.Handle(pat.Post("/user/new"), Handler{env: &Env{db: db}, handler: newUserHandler})
}

// newUserHandler is the handler function for the new user registration operations
func newUserHandler(env *Env, w http.ResponseWriter, r *http.Request) error {
	err := env.db.SaveUser(r.Context(), "foobar")
	if err != nil {
		return &StatusError{
			Code: http.StatusUnprocessableEntity,
			Msg:  err.Error(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("ok"))

	return nil
}
