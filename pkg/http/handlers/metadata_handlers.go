package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/pkg/errors"
	"github.com/thingful/kuzu/pkg/postgres"
	goji "goji.io"
	"goji.io/pat"
)

// RegisterMetadataHandlers registers any handlers for metadata operations
func RegisterMetadataHandlers(mux *goji.Mux, db *postgres.DB) {
	mux.Handle(pat.Post("/api/entity/timeSeriesInformations/get"), Handler{env: &Env{db: db}, handler: metadataHandler})
}

// timeseries is a struct used when rendering output for the metadata endpoint
type timeseries struct {
	ID           int64  `json:"TimeSeriesInformationId"`
	LocationID   string `json:"LocationIdentifier"`
	DataSourceID int64  `json:"DataSourceVariableId"`
	StartDate    string `json:"StartDate"`
	EndDate      string `json:"EndDate"`
}

func metadataHandler(env *Env, w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	metadata, err := env.db.GetMetadata(ctx)
	if err != nil {
		return &HTTPError{
			Code: http.StatusInternalServerError,
			Err:  errors.Wrap(err, "failed to read metadata from db"),
		}
	}

	ts := map[string]timeseries{}

	for _, m := range metadata {
		t := timeseries{
			ID:           m.ID,
			LocationID:   fmt.Sprintf("Grow.Thingful.%s", m.ThingUID),
			DataSourceID: m.DataSourceID,
			StartDate:    m.FirstSampleUTC.Time.Format("20060102150405"),
			EndDate:      m.LastSampleUTC.Time.Format("20060102150405"),
		}

		ts[strconv.FormatInt(m.ID, 10)] = t
	}

	b, err := json.Marshal(struct {
		TimeSeriesInformation map[string]timeseries `json:"TimeSeriesInformations"`
	}{
		TimeSeriesInformation: ts,
	})
	if err != nil {
		return &HTTPError{
			Code: http.StatusInternalServerError,
			Err:  errors.Wrap(err, "failed to marshal output json"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)

	return nil
}
