package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/guregu/null"
	"github.com/pkg/errors"
	"github.com/thingful/kuzu/pkg/flowerpower"
	"github.com/thingful/kuzu/pkg/postgres"
	goji "goji.io"
	"goji.io/pat"
)

// RegisterLocationHandlers registers handlers for working with locations
func RegisterLocationHandlers(mux *goji.Mux, db *postgres.DB, th Thingful) {
	mux.Handle(pat.Post("/api/entity/locations/get"), Handler{env: &Env{db: db}, handler: listLocationsHandler})
	mux.Handle(pat.Post("/api/entity/locations/update"), Handler{env: &Env{db: db, thingful: th}, handler: updateLocationHandler})
}

// listLocationsRequest is used to parse incoming requests for locations
type listLocationsRequest struct {
	UserUID         string   `json:"UserId"`
	DataSourceCodes []string `json:"DataSourceCodes"`
	InvalidLocation bool     `json:"InvalidLocation"`
	StaleData       bool     `json:"StaleData"`
}

// updateLocationRequest is used to parse incoming requests to set the location
// of a device
type updateLocationRequest struct {
	Code string  `json:"Code"`
	X    float64 `json:"X"`
	Y    float64 `json:"Y"`
}

// location is used when rendering the response to the client. The structure is
// defined by hydronet
type location struct {
	Code                         string  `json:"Code"`
	DataSourceGroupCode          string  `json:"DataSourceGroupCode"`
	Identifier                   string  `json:"Identifier"`
	Name                         string  `json:"Name"`
	LocationID                   int64   `json:"LocationId"`
	ProjectionID                 int64   `json:"ProjectionId"`
	X                            float64 `json:"X"`
	Y                            float64 `json:"Y"`
	Z                            float64 `json:"Z"`
	FirstSampleTimestamp         string  `json:"FirstSampleTimestamp"`
	LastAvailableSampleTimestamp string  `json:"LastAvailableSampleTimestamp"`
	LastFetchedSampleTimestamp   string  `json:"LastFetchedSampleTimestamp"`
}

// listLocationsHandler is our handler that returns location information to
// clients
func listLocationsHandler(env *Env, w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	req, err := parseListRequest(r)
	if err != nil {
		return err
	}

	locations, err := env.db.ListLocations(ctx, req.UserUID, req.InvalidLocation, req.StaleData)
	if err != nil {
		return &HTTPError{
			Code: http.StatusInternalServerError,
			Err:  errors.Wrap(err, "failed to list locations"),
		}
	}

	locationMap := map[string]*location{}

	for _, loc := range locations {
		l := buildLocation(&loc)
		locationMap[fmt.Sprintf("Grow.Thingful#%s", loc.UID)] = l
	}

	b, err := json.Marshal(struct {
		Locations map[string]*location `json:"Locations"`
	}{
		Locations: locationMap,
	})
	if err != nil {
		return &HTTPError{
			Code: http.StatusInternalServerError,
			Err:  errors.Wrap(err, "failed to marshal response JSON"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)

	return nil
}

func parseListRequest(r *http.Request) (*listLocationsRequest, error) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, &HTTPError{
			Code: http.StatusInternalServerError,
			Err:  errors.Wrap(err, "failed to read incoming request body"),
		}
	}

	var data listLocationsRequest
	err = json.Unmarshal(b, &data)
	if err != nil {
		return nil, &HTTPError{
			Code: http.StatusUnprocessableEntity,
			Err:  errors.Wrap(err, "failed to parse incoming request body"),
		}
	}

	return &data, nil
}

func updateLocationHandler(env *Env, w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	req, err := parseUpdateRequest(r)
	if err != nil {
		return err
	}

	loc, err := env.db.UpdateGeolocation(ctx, req.Code, req.X, req.Y)
	if err != nil {
		return &HTTPError{
			Code: http.StatusInternalServerError,
			Err:  errors.Wrap(err, "failed to update geolocation"),
		}
	}

	err = env.thingful.UpdateThing(ctx, &postgres.Thing{
		UID:        null.StringFrom(loc.UID),
		LocationID: loc.LocationID,
		Nickname:   null.StringFrom(loc.Nickname),
		Longitude:  loc.Longitude,
		Latitude:   loc.Latitude,
	}, []flowerpower.Reading{})
	if err != nil {
		return &HTTPError{
			Code: http.StatusInternalServerError,
			Err:  err,
		}
	}

	output := buildLocation(loc)

	b, err := json.Marshal(output)
	if err != nil {
		return &HTTPError{
			Code: http.StatusInternalServerError,
			Err:  errors.Wrap(err, "failed to marshal response JSON"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)

	return nil
}

// parseUpdateRequest builds a updateLocationRequest object or returns an error
func parseUpdateRequest(r *http.Request) (*updateLocationRequest, error) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, &HTTPError{
			Code: http.StatusInternalServerError,
			Err:  errors.Wrap(err, "failed to read incoming request body"),
		}
	}

	var data updateLocationRequest
	err = json.Unmarshal(b, &data)
	if err != nil {
		return nil, &HTTPError{
			Code: http.StatusUnprocessableEntity,
			Err:  errors.Wrap(err, "failed to parse incoming request body"),
		}
	}

	return &data, nil
}

// buildLocation builds our output location type from the location returned from
// Postgres
func buildLocation(loc *postgres.Location) *location {
	return &location{
		Code:                         loc.UID,
		DataSourceGroupCode:          "Grow.Thingful",
		Identifier:                   fmt.Sprintf("Grow.Thingful#%s", loc.UID),
		Name:                         fmt.Sprintf("%s. Serial Number: %s", loc.Nickname, loc.SerialNum),
		LocationID:                   loc.ID,
		ProjectionID:                 3,
		X:                            loc.Longitude,
		Y:                            loc.Latitude,
		Z:                            0,
		FirstSampleTimestamp:         loc.FirstSampleUTC.Time.Format("20060102150405"),
		LastAvailableSampleTimestamp: loc.LastSampleUTC.Time.Format("20060102150405"),
		LastFetchedSampleTimestamp:   loc.LastUploadedSampleUTC.Time.Format("20060102150405"),
	}
}
