package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
	"time"

	goji "goji.io"
	"goji.io/pat"

	"github.com/pkg/errors"
	"github.com/thingful/kuzu/pkg/postgres"
	"github.com/thingful/kuzu/pkg/thingful"
)

const (
	maxTimeInterval = 10

	maxLocations = 10

	timeFormat = "20060102150405"
)

// RegisterTimeseriesHandler registers our handler that requests data from Thingful
func RegisterTimeseriesHandler(mux *goji.Mux, db *postgres.DB, th *thingful.Thingful) {
	mux.Handle(pat.Post("/timeSeries/get"), Handler{env: &Env{db: db, thingful: th}, handler: timeseriesHandler})
}

type setting struct {
	LocationCodes   []string  `json:"LocationCodes"`
	VariableCodes   []string  `json:"VariableCodes"`
	StartDate       time.Time `json:"StartDate"`
	EndDate         time.Time `json:"EndDate"`
	Ascending       bool      `json:"Order"`
	StructureType   string    `json:"StructureType"`
	CalculationType string    `json:"CalculationType"`
}

// UnmarshalJSON is a custom unmarshaller to convert strings into times and
// boolean values
func (s *setting) UnmarshalJSON(data []byte) error {
	type aliasedSetting setting

	set := &struct {
		StartDate string `json:"StartDate"`
		EndDate   string `json:"EndDate"`
		Ascending string `json:"Order"`
		*aliasedSetting
	}{
		aliasedSetting: (*aliasedSetting)(s),
	}

	err := json.Unmarshal(data, &set)
	if err != nil {
		return err
	}

	s.StartDate, err = time.Parse(timeFormat, set.StartDate)
	if err != nil {
		return err
	}

	s.EndDate, err = time.Parse(timeFormat, set.EndDate)
	if err != nil {
		return err
	}

	s.Ascending = strings.ToLower(set.Ascending) == "asc"

	return nil
}

type reader struct {
	DataSourceCode string  `json:"DataSourceCode"`
	Setting        setting `json:"Settings"`
}

type timeseriesRequest struct {
	Readers []reader `json:"Readers"`
}

type observation struct {
	Value        float64 `json:"Value"`
	DateTime     time.Time
	Availability int `json:"Availability"`
	Quality      int `json:"Quality"`
}

func (o observation) MarshalJSON() ([]byte, error) {
	type O observation

	return json.Marshal(&struct {
		DateTime string `json:"DateTime"`
		O
	}{
		DateTime: o.DateTime.Format(timeFormat),
		O:        (O)(o),
	})
}

type series struct {
	StartDate            time.Time
	EndDate              time.Time
	LocationIdentifier   string        `json:"LocationIdentifier"`
	LocationCode         string        `json:"LocationCode"`
	Data                 []observation `json:"Data"`
	VariableCode         string        `json:"VariableCode"`
	DataSourceVariableID int           `json:"DataSourceVariableId"`
	SensorName           string        `json:"SensorName"`
	SerialNumber         string        `json:"SerialNumber"`
}

type timeseriesResponse struct {
	Data []series `json:"Data"`
}

func timeseriesHandler(env *Env, w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	rd, err := parseTimeSeriesRequest(r)
	if err != nil {
		return err
	}

	things, err := env.thingful.GetData(ctx, rd.LocationCodes, rd.StartDate, rd.EndDate, rd.Ascending)
	if err != nil {
		return &HTTPError{
			Code: http.StatusInternalServerError,
			Err:  errors.Wrap(err, "failed to get data from Thingful"),
		}
	}

	resp, err := buildResponse(things)
	if err != nil {
		return &HTTPError{
			Code: http.StatusInternalServerError,
			Err:  errors.Wrap(err, "failed to build response"),
		}
	}

	b, err := json.Marshal(resp)
	if err != nil {
		return &HTTPError{
			Code: http.StatusInternalServerError,
			Err:  errors.Wrap(err, "failed to marshal response"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)

	return nil
}

func parseTimeSeriesRequest(r *http.Request) (*setting, error) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, &HTTPError{
			Code: http.StatusInternalServerError,
			Err:  errors.Wrap(err, "failed to read incoming request body"),
		}
	}

	var data timeseriesRequest
	err = json.Unmarshal(b, &data)
	if err != nil {
		return nil, &HTTPError{
			Code: http.StatusUnprocessableEntity,
			Err:  errors.Wrap(err, "failed to parse incoming request bodey"),
		}
	}

	// validate the query
	if len(data.Readers) != 1 {
		return nil, &HTTPError{
			Code: http.StatusUnprocessableEntity,
			Err:  errors.New("Invalid readers count - must be a single Reader element only"),
		}
	}

	reader := data.Readers[0]
	if reader.DataSourceCode != "Thingful.Connectors.GROWSensors" {
		return nil, &HTTPError{
			Code: http.StatusUnprocessableEntity,
			Err:  fmt.Errorf("unexpected DataSourceCode, expected Thingful.Connectors.GROWSensor, got %s", reader.DataSourceCode),
		}
	}

	if len(reader.Setting.LocationCodes) == 0 {
		return nil, &HTTPError{
			Code: http.StatusUnprocessableEntity,
			Err:  errors.New("missing location identifier"),
		}
	}

	if len(reader.Setting.LocationCodes) > maxLocations {
		return nil, &HTTPError{
			Code: http.StatusUnprocessableEntity,
			Err:  errors.New("too many location identifiers, max permitted is 10"),
		}
	}

	if reader.Setting.StartDate.IsZero() || reader.Setting.EndDate.IsZero() {
		return nil, &HTTPError{
			Code: http.StatusUnprocessableEntity,
			Err:  errors.New("must supply a valid start and end date"),
		}
	}

	if reader.Setting.StartDate.After(reader.Setting.EndDate) {
		return nil, &HTTPError{
			Code: http.StatusUnprocessableEntity,
			Err:  errors.New("start date must be before end date"),
		}
	}

	timeRange := reader.Setting.EndDate.Sub(reader.Setting.StartDate)
	if timeRange.Hours()/24 > maxTimeInterval {
		return nil, &HTTPError{
			Code: http.StatusUnprocessableEntity,
			Err:  fmt.Errorf("maximum permitted time interval is %v days", maxTimeInterval),
		}
	}

	return &reader.Setting, nil
}

func buildResponse(things []thingful.Thing) (*timeseriesResponse, error) {
	allSeries := []series{}

	for _, t := range things {
		uid := path.Base(t.ID)
		serialNum := t.Attributes.Metadata[0].Val

		s := series{
			LocationIdentifier: fmt.Sprintf("Grow.Thingful#%s", uid),
			LocationCode:       uid,
			SensorName:         t.Attributes.Title,
			SerialNumber:       serialNum,
		}

		allSeries = append(allSeries, s)
	}

	return &timeseriesResponse{
		Data: allSeries,
	}, nil
}
