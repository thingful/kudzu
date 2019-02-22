package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/thingful/kuzu/pkg/postgres"
	goji "goji.io"
	"goji.io/pat"
)

const (
	nodeName = "Thingful.Connectors.GROWSensors"
)

// RegisterDataSourceHandlers registers our data source related handlers
func RegisterDataSourceHandlers(mux *goji.Mux, db *postgres.DB) {
	mux.Handle(pat.Post("/api/entity/dataSourceVariables/get"), Handler{env: &Env{db: db}, handler: datasourcesHandler})
}

// datasource is our type used for generating datasource output
type datasource struct {
	ID               int64  `json:"DataSourceVariableId"`
	Variable         string `json:"VariableCode"`
	DataSourceCode   string `json:"DataSourceCode"`
	Name             string `json:"Name"`
	Code             string `json:"Code"`
	UnitCode         string `json:"UnitCode"`
	DataType         string `json:"DataType"`
	MathematicalType string `json:"MathematicalType"`
	MeasurementType  string `json:"MeasurementType"`
	State            int64  `json:"State"`
	Cumulative       bool   `json:"IsCumulative"`
}

func datasourcesHandler(env *Env, w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	datasources, err := env.db.GetDataSources(ctx)
	if err != nil {
		return &HTTPError{
			Code: http.StatusInternalServerError,
			Err:  err,
		}
	}

	variables := map[string]datasource{}

	for _, ds := range datasources {
		d := datasource{
			ID:               ds.ID,
			Variable:         fmt.Sprintf("%s.%s", nodeName, ds.Code),
			DataSourceCode:   nodeName,
			Name:             strings.Title(strings.Replace(ds.Code, "_", " ", -1)),
			Code:             ds.Code,
			UnitCode:         unitToHN4(ds.Code, ds.Unit),
			DataType:         dataTypeToHN4(ds.DataType),
			MathematicalType: "NotSummable",
			MeasurementType:  "Instantaneous",
			State:            1,
			Cumulative:       false,
		}

		variables[strconv.FormatInt(ds.ID, 10)] = d
	}

	b, err := json.Marshal(struct {
		Variables map[string]datasource `json:"DataSourceVariables"`
	}{
		Variables: variables,
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
