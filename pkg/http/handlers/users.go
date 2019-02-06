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
	"github.com/thingful/kuzu/pkg/postgres"
)

// RegisterUserHandlers registers our user related handlers into the mux
func RegisterUserHandlers(mux *goji.Mux, db *postgres.DB, cl *client.Client) {
	mux.Handle(pat.Post("/user/new"), Handler{env: &Env{db: db, client: cl}, handler: newUserHandler})
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

	userData, err := parseNewUserRequest(r)
	if err != nil {
		return err
	}

	sensorCount, err := flowerpower.SensorCount(env.client, userData.Info.AccessToken)
	if err != nil {
		return &HTTPError{
			Code: http.StatusBadGateway,
			Err:  errors.Wrap(err, "failed to count sensors from flowerpower api"),
		}
	}

	// save the user with identity
	//err = env.db.SaveUser(r.Context(), userData.Info.UID, userData.Info.AccessToken, userData.Info.RefreshToken, userData.Info.Provider)
	//if err != nil {
	//	return &HTTPError{
	//		Code: http.StatusInternalServerError,
	//		Err:  errors.Wrap(err, "failed to save user"),
	//	}
	//}

	// build response
	b, err := json.Marshal(struct {
		UserUID     string `json:"User"`
		TotalThings int    `json:"TotalThings"`
	}{
		UserUID:     userData.Info.UID,
		TotalThings: sensorCount,
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

	// spawn process to index all sensors for the new user

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

	if data.Info.UID == "" || data.Info.AccessToken == "" {
		return nil, &HTTPError{
			Code: http.StatusUnprocessableEntity,
			Err:  errors.New("User identifier and access token must be supplied"),
		}
	}

	return &data, nil
}
