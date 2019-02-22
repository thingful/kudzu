package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	goji "goji.io"
	"goji.io/pat"

	"github.com/thingful/kuzu/pkg/client"
	"github.com/thingful/kuzu/pkg/flowerpower"
	"github.com/thingful/kuzu/pkg/indexer"
	"github.com/thingful/kuzu/pkg/logger"
	"github.com/thingful/kuzu/pkg/postgres"
)

// RegisterUserHandlers registers our user related handlers into the mux
func RegisterUserHandlers(mux *goji.Mux, db *postgres.DB, cl *client.Client, in *indexer.Indexer) {
	mux.Handle(pat.Post("/user/new"), Handler{env: &Env{db: db, client: cl, indexer: in}, handler: newUserHandler})
	mux.Handle(pat.Delete("/user/delete"), Handler{env: &Env{db: db}, handler: deleteUserHandler})
}

// newUserRequest is a local type used for parsing incoming requests
type newUserRequest struct {
	Info struct {
		UID          string `json:"Identifier"`
		Provider     string `json:"Provider"`
		AccessToken  string `json:"AccessToken"`
		RefreshToken string `json:"RefreshToken"`
	} `json:"User"`
}

// newUserHandler is the handler function for the new user registration operations
func newUserHandler(env *Env, w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	log := logger.FromContext(ctx)

	// parse the incoming request
	userData, err := parseNewUserRequest(r)
	if err != nil {
		return err
	}

	if userData.Info.UID == "" || userData.Info.AccessToken == "" {
		return &HTTPError{
			Code: http.StatusUnprocessableEntity,
			Err:  errors.New("User identifier and access token must be supplied"),
		}
	}

	// get user profile from parrot
	parrotUser, err := flowerpower.GetUser(ctx, env.client, userData.Info.AccessToken)
	if err != nil {
		return &HTTPError{
			Code: http.StatusBadGateway,
			Err:  errors.New("failed to read profile information from flowerpower API"),
		}
	}

	// save the user with identity to postgres
	userID, err := env.db.SaveUser(ctx, &postgres.User{
		UID:          userData.Info.UID,
		ParrotID:     parrotUser.ParrotID,
		AccessToken:  userData.Info.AccessToken,
		RefreshToken: userData.Info.RefreshToken,
		Provider:     userData.Info.Provider,
	})
	if err != nil {
		switch errors.Cause(err) {
		case postgres.ClientError:
			return &HTTPError{
				Code: http.StatusUnprocessableEntity,
				Err:  err,
			}
		default:
			return &HTTPError{
				Code: http.StatusInternalServerError,
				Err:  err,
			}
		}
	}

	log.Log(
		"msg", "created user record",
		"userID", userID,
	)

	// get locations from parrot
	locations, err := flowerpower.GetLocations(ctx, env.client, userData.Info.AccessToken)
	if err != nil {
		return &HTTPError{
			Code: http.StatusBadGateway,
			Err:  errors.Wrap(err, "failed to read locations from flowerpower API"),
		}
	}

	// build response
	b, err := json.Marshal(struct {
		UserUID     string `json:"User"`
		TotalThings int    `json:"TotalThings"`
	}{
		UserUID:     userData.Info.UID,
		TotalThings: len(locations),
	})
	if err != nil {
		return &HTTPError{
			Code: http.StatusInternalServerError,
			Err:  errors.Wrap(err, "failed to marshal response JSON"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write(b)

	return nil
}

// parseNewUserRequest attempts to parse from the incoming request a new user
// request object containing the incoming data. Here we also perform some basic
// validation to make sure required fields are set.
func parseNewUserRequest(r *http.Request) (*newUserRequest, error) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, &HTTPError{
			Code: http.StatusInternalServerError,
			Err:  errors.Wrap(err, "failed to read incoming request body"),
		}
	}

	var data newUserRequest

	err = json.Unmarshal(b, &data)
	if err != nil {
		return nil, &HTTPError{
			Code: http.StatusUnprocessableEntity,
			Err:  errors.Wrap(err, "failed to parse incoming request body"),
		}
	}

	return &data, nil
}

func deleteUserHandler(env *Env, w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	log := logger.FromContext(ctx)

	userData, err := parseNewUserRequest(r)
	if err != nil {
		return err
	}

	if userData.Info.UID == "" {
		return &HTTPError{
			Code: http.StatusUnprocessableEntity,
			Err:  errors.New("User identifier must be supplied"),
		}
	}

	err = env.db.DeleteUser(ctx, userData.Info.UID)
	if err != nil {
		log.Log(
			"msg", "error deleting user",
			"error", err,
		)
		switch errors.Cause(err) {
		case postgres.ClientError:
			return &HTTPError{
				Code: http.StatusNotFound,
				Err:  errors.New("Unable to delete the specified user"),
			}
		default:
			return &HTTPError{
				Code: http.StatusInternalServerError,
				Err:  err,
			}
		}
	}

	w.WriteHeader(http.StatusNoContent)

	return nil
}
