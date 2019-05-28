package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	goji "goji.io"
	"goji.io/pat"

	"github.com/guregu/null"
	"github.com/pkg/errors"
	"github.com/thingful/kudzu/pkg/logger"
	"github.com/thingful/kudzu/pkg/postgres"
	"github.com/thingful/kudzu/pkg/thingful"
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

// setting is a type used to parse an incoming timeseries request. We build the
// object, then fetch the appropriate data from the Thingful backend. We then
// filter the returned data to only include the requested VariableCodes before
// returning to the client.
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

// output types

// observation is used to show a single value for a variable
type observation struct {
	Value    float64 `json:"Value"`
	DateTime time.Time
}

// MarshalJSON is an implementation of the marshaller interface to add extra
// fields on serialization
func (o observation) MarshalJSON() ([]byte, error) {
	type O observation

	return json.Marshal(&struct {
		DateTime     string `json:"DateTime"`
		Availability int    `json:"Availability"`
		Quality      int    `json:"Quality"`
		O
	}{
		DateTime:     o.DateTime.Format(timeFormat),
		Availability: 1,
		Quality:      0,
		O:            (O)(o),
	})
}

// series represents a single channels data for the requested time window
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

// MarshalJSON is an implementation of the marshaller interface to add extra
// fields on serialization
func (s series) MarshalJSON() ([]byte, error) {
	type S series

	type interval struct {
		Type  string
		Value int
	}

	return json.Marshal(&struct {
		StartDate       string   `json:"StartDate"`
		EndDate         string   `json:"EndDate"`
		DataType        string   `json:"DataType"`
		TimeZoneOffset  string   `json:"TimeZoneOffset"`
		IsCumulative    bool     `json:"IsCumulative"`
		UseQuality      bool     `json:"UseQuality"`
		CalculationType string   `json:"CalculationType"`
		NoDataValue     int      `json:"NoDataValue"`
		Interval        interval `json:"Interval"`
		S
	}{
		StartDate:       s.StartDate.Format(timeFormat),
		EndDate:         s.EndDate.Format(timeFormat),
		DataType:        "Double",
		TimeZoneOffset:  "+0000",
		IsCumulative:    false,
		CalculationType: "None",
		UseQuality:      false,
		NoDataValue:     -9999,
		Interval: interval{
			Type:  "None",
			Value: 0,
		},
		S: (S)(s),
	})
}

// timeseriesResponse is the container struct we return to clients requesting
// time series data.
type timeseriesResponse struct {
	Data []series `json:"Data"`
	Meta meta     `json:"Meta"`
}

type meta struct {
	Locations           map[string]hydronetLocation           `json:"Locations"`
	Units               map[string]hydronetUnit               `json:"Units"`
	DataSourceVariables map[string]hydronetDatasourceVariable `json:"DataSourceVariables"`
	Variables           map[string]hydronetVariable           `json:"Variables"`
}

func (m meta) MarshalJSON() ([]byte, error) {
	type projection struct {
		ProjectionID     int `json:"ProjectionId"`
		Name             string
		Epsg             int
		ProjectionString string
	}

	type M meta
	return json.Marshal(&struct {
		Projections map[string]projection
		M
	}{
		Projections: map[string]projection{
			"3": projection{
				ProjectionID:     3,
				Name:             "WGS84",
				Epsg:             4326,
				ProjectionString: "+proj=longlat +ellps=WGS84 +datum=WGS84 +no_defs ",
			},
		},
		M: (M)(m),
	})
}

// hydronetLocation is a struct used for serialising data to HydroNet.
type hydronetLocation struct {
	Identifier   string
	Name         string
	Code         string
	SerialNumber string
	X            float64
	Y            float64
	Z            float64
}

// MarshalJSON is our custom marshalling function that adds some extra fields.
func (l hydronetLocation) MarshalJSON() ([]byte, error) {
	type L hydronetLocation
	return json.Marshal(&struct {
		DataSourceGroupCode string
		ProjectionID        int `json:"ProjectionId"`
		L
	}{
		DataSourceGroupCode: "Grow.Thingful",
		ProjectionID:        3,
		L:                   (L)(l),
	})
}

// hydronetUnit is a struct used for serialising data to HydroNet.
type hydronetUnit struct {
	Name string
	Code string
}

type hydronetDatasourceVariable struct {
	ID               int `json:"DataSourceVariableId"`
	Code             string
	Name             string
	VariableCode     string
	DataSourceCode   string
	UnitCode         string
	DataType         string
	MathematicalType string
	MeasurementType  string
	State            int
	IsCumulative     bool
}

type hydronetVariable struct {
	State      int
	VariableID int `json:"VariableId"`
	Name       string
	Code       string
	UnitCode   string
}

// timeSeriesHandler is the handler that handles time series requests
func timeseriesHandler(env *Env, w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	log := logger.FromContext(ctx)

	// parse the request giving back a setting to read data for
	rd, err := parseTimeSeriesRequest(r)
	if err != nil {
		log.Log("error", err, "msg", "failed to parse request")
		return err
	}

	// get a slice of things from Thingful for the requested locations
	things, err := env.thingful.GetData(ctx, rd.LocationCodes, rd.StartDate, rd.EndDate, rd.Ascending)
	if err != nil {
		return &HTTPError{
			Code: http.StatusInternalServerError,
			Err:  errors.Wrap(err, "failed to get data from Thingful"),
		}
	}

	datasources, err := env.db.GetDataSources(ctx)
	if err != nil {
		return &HTTPError{
			Code: http.StatusInternalServerError,
			Err:  errors.Wrap(err, "failed to read datasources from the DB"),
		}
	}

	resp, err := buildResponse(things, rd.VariableCodes, datasources, rd.Ascending)
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

	fmt.Println(string(b))
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

func buildResponse(things []thingful.Thing, variableCodes []string, datasources []postgres.DataSource, ascending bool) (*timeseriesResponse, error) {
	allSeries := []series{}
	locations := map[string]hydronetLocation{}
	units := map[string]hydronetUnit{}
	datasourceVariables := map[string]hydronetDatasourceVariable{}
	variables := map[string]hydronetVariable{}

	for _, t := range things {
		// some attributes will be the same for every series of a thing
		uid := path.Base(t.ID)
		serialNum := getSerialNum(t.Attributes.Metadata)
		locationIdentifier := fmt.Sprintf("Grow.Thingful#%s", uid)

		// add location metadata
		locations[uid] = hydronetLocation{
			Identifier:   locationIdentifier,
			Name:         fmt.Sprintf("%s. Serial Number: %s", t.Attributes.Title, serialNum),
			Code:         uid,
			SerialNumber: serialNum,
			X:            t.Attributes.Location.Longitude,
			Y:            t.Attributes.Location.Latitude,
		}

		for _, c := range t.Attributes.Channels {
			// skip any channel with no observations as other stuff will fail in this
			// case
			if len(c.Observations) == 0 {
				continue
			}

			// check if this variable code has been requested - skip to the next if not
			variableCode := getVariableCode(c.ID)
			if !isPresent(variableCode, variableCodes) {
				continue
			}

			// build units
			unitKey := getUnitKey(c.ID, c.Unit)
			if _, ok := units[unitKey]; !ok {
				if unitKey != "" {
					units[unitKey] = buildUnit(unitKey)
				}
			}

			// build datasourcevariables
			datasource := getDatasource(c.ID, datasources)
			if datasource != nil {
				datasourceVariableKey := strconv.FormatInt(datasource.ID, 10)
				if _, ok := datasourceVariables[datasourceVariableKey]; !ok {
					if datasourceVariableKey != "" {
						datasourceVariables[datasourceVariableKey] = buildDatasourceVariable(datasource, unitKey)
					}
				}
			}

			// build variables
			if _, ok := variables[variableCode]; !ok {
				variables[variableCode] = buildVariable(c.ID, variableCode, unitKey)
			}

			observations, err := buildObservations(c.Observations, ascending)
			if err != nil {
				return nil, err
			}

			var (
				startDate, endDate time.Time
			)

			// build observations
			if ascending {
				startDate = observations[0].DateTime
				endDate = observations[len(observations)-1].DateTime
			} else {
				startDate = observations[len(observations)-1].DateTime
				endDate = observations[0].DateTime
			}

			s := series{
				LocationIdentifier: locationIdentifier,
				LocationCode:       uid,
				SensorName:         t.Attributes.Title,
				SerialNumber:       serialNum,
				StartDate:          startDate,
				EndDate:            endDate,
				VariableCode:       variableCode,
				Data:               observations,
			}

			if datasource != nil {
				s.DataSourceVariableID = int(datasource.ID)
			}

			allSeries = append(allSeries, s)
		}
	}

	return &timeseriesResponse{
		Data: allSeries,
		Meta: meta{
			Locations:           locations,
			Units:               units,
			DataSourceVariables: datasourceVariables,
			Variables:           variables,
		},
	}, nil
}

// buildVariableCode takes as input the channel ID from Thingful, and returns
// the HydroNet variable code format.
func getVariableCode(id string) string {
	return fmt.Sprintf("Thingful.Connectors.GROWSensors.%s", path.Base(id))
}

// isPresent if the passed in string is within the given slice (i.e. the
// haystack)
func isPresent(needle string, haystack []string) bool {
	for _, elem := range haystack {
		if needle == elem {
			return true
		}
	}
	return false
}

// buildObservations returns a slice of output observations from the data
// received from Thingful. Returns an error if any value is unable to be parsed
// as a float.
func buildObservations(input []thingful.Observation, ascending bool) ([]observation, error) {
	observations := []observation{}

	if ascending {
		for i := len(input) - 1; i >= 0; i-- {
			val, err := strconv.ParseFloat(input[i].Value, 64)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse value to a float")
			}

			o := observation{
				Value:    val,
				DateTime: input[i].RecordedAt,
			}

			observations = append(observations, o)
		}
	} else {
		for _, i := range input {
			val, err := strconv.ParseFloat(i.Value, 64)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse value to a float")
			}

			o := observation{
				Value:    val,
				DateTime: i.RecordedAt,
			}

			observations = append(observations, o)
		}
	}

	return observations, nil
}

func getUnitKey(channelID, unit string) string {
	return unitToHN4(path.Base(channelID), null.NewString(unit, unit != ""))
}

func buildUnit(unitKey string) hydronetUnit {
	return hydronetUnit{
		Name: unitKey,
		Code: unitKey,
	}
}

func getDatasource(channelID string, datasources []postgres.DataSource) *postgres.DataSource {
	name := path.Base(channelID)
	for _, datasource := range datasources {
		if datasource.Code == name {
			return &datasource
		}
	}
	return nil
}

func buildDatasourceVariable(ds *postgres.DataSource, unitKey string) hydronetDatasourceVariable {
	return hydronetDatasourceVariable{
		ID:               int(ds.ID),
		Code:             ds.Code,
		Name:             titleizeSnake(ds.Code),
		VariableCode:     fmt.Sprintf("Thingful.Connectors.GROWSensors.%s", ds.Code),
		DataSourceCode:   "Thingful.Connectors.GROWSensors",
		UnitCode:         unitKey,
		DataType:         "Double",
		MathematicalType: "NotSummable",
		MeasurementType:  "Instantaneous",
		State:            1,
		IsCumulative:     false,
	}
}

func buildVariable(channelID, variableCode, unitKey string) hydronetVariable {
	return hydronetVariable{
		State:      1,
		VariableID: 0,
		Name:       pascalizeSnake(path.Base(channelID)),
		Code:       variableCode,
		UnitCode:   unitKey,
	}
}

func getSerialNum(metadata []thingful.Metadata) string {
	for _, m := range metadata {
		if m.Prop == "http://schema.org/serialNumber" {
			return m.Val
		}
	}
	return ""
}

func titleizeSnake(input string) string {
	return strings.Title(strings.Replace(input, "_", " ", -1))
}

func pascalizeSnake(input string) string {
	return strings.Replace(titleizeSnake(input), " ", "", -1)
}
