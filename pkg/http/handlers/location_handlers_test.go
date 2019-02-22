package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/thingful/kuzu/pkg/client"
	"github.com/thingful/kuzu/pkg/http/handlers"
	"github.com/thingful/kuzu/pkg/logger"
	"github.com/thingful/kuzu/pkg/postgres"
	"github.com/thingful/kuzu/pkg/postgres/helper"
	"github.com/thingful/kuzu/pkg/thingful"
	"github.com/thingful/simular"
	goji "goji.io"
)

type LocationHandlersSuite struct {
	suite.Suite
	db       *postgres.DB
	logger   kitlog.Logger
	client   *client.Client
	thingful *thingful.Thingful
}

func (s *LocationHandlersSuite) SetupTest() {
	logger := kitlog.NewNopLogger()
	connStr := os.Getenv("KUZU_DATABASE_URL")

	s.db = helper.PrepareDB(s.T(), connStr, logger)
	s.logger = logger
	s.client = client.NewClient(1, true)
	s.thingful = thingful.NewClient(s.client, "http://thingful.net", "api-key", true, 2)
}

func (s *LocationHandlersSuite) TearDownTest() {
	helper.CleanDB(s.T(), s.db)
}

func (s *LocationHandlersSuite) TestListLocations() {
	var userID int64
	err := s.db.DB.Get(&userID, `INSERT INTO users (uid) VALUES ($1) RETURNING id`, "alice")
	assert.Nil(s.T(), err)

	var userID2 int64
	err = s.db.DB.Get(&userID2, `INSERT INTO users (uid) VALUES ($1) RETURNING id`, "bob")
	assert.Nil(s.T(), err)

	ctx := logger.ToContext(context.Background(), s.logger)

	// insert some things
	_, err = s.db.DB.Exec(`
		INSERT INTO things (uid, owner_id, serial_num, long, lat, location_identifier, last_sample)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())`, "1234", userID, "PA1", 12.2, 13.3, "LOC1",
	)
	assert.Nil(s.T(), err)
	_, err = s.db.DB.Exec(`
		INSERT INTO things (uid, owner_id, serial_num, long, lat, location_identifier, last_sample)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())`, "1235", userID2, "PA2", 0, 0, "LOC2",
	)
	assert.Nil(s.T(), err)
	_, err = s.db.DB.Exec(`
		INSERT INTO things (uid, owner_id, serial_num, long, lat, location_identifier, last_sample)
		VALUES ($1, $2, $3, $4, $5, $6, NOW() - interval '31 days')`, "1236", userID, "PA3", 12.2, 13.3, "LOC3",
	)
	assert.Nil(s.T(), err)

	mux := goji.NewMux()
	handlers.RegisterLocationHandlers(mux, s.db, s.thingful)

	testcases := []struct {
		label           string
		requestBody     []byte
		expectedLength  int
		expectedFirstID string
	}{
		{
			label:           "all locations",
			requestBody:     []byte(`{"DataSourceCodes":["Thingful.Connectors.GROWSensors"]}`),
			expectedLength:  3,
			expectedFirstID: "Grow.Thingful#1234",
		},
		{
			label:           "just alice locations",
			requestBody:     []byte(`{"DataSourceCodes":["Thingful.Connectors.GROWSensors"],"UserId":"alice"}`),
			expectedLength:  2,
			expectedFirstID: "Grow.Thingful#1234",
		},
		{
			label:           "invalid geolocations",
			requestBody:     []byte(`{"DataSourceCodes":["Thingful.Connectors.GROWSensors"],"InvalidLocation":true}`),
			expectedLength:  1,
			expectedFirstID: "Grow.Thingful#1235",
		},
		{
			label:           "stale data",
			requestBody:     []byte(`{"DataSourceCodes":["Thingful.Connectors.GROWSensors"],"StaleData":true}`),
			expectedLength:  1,
			expectedFirstID: "Grow.Thingful#1236",
		},
	}

	for _, tc := range testcases {
		s.T().Run(tc.label, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPost, "/entity/locations/get", bytes.NewReader(tc.requestBody))
			req = req.WithContext(ctx)
			assert.Nil(t, err)

			mux.ServeHTTP(recorder, req)
			assert.Equal(t, http.StatusOK, recorder.Code)

			var parsedResp map[string]interface{}

			err = json.Unmarshal(recorder.Body.Bytes(), &parsedResp)
			assert.Nil(t, err)
			assert.Len(t, parsedResp["Locations"], tc.expectedLength)
			assert.NotNil(t, parsedResp["Locations"].(map[string]interface{})[tc.expectedFirstID])
		})
	}
}

func (s *LocationHandlersSuite) TestUpdatelocation() {
	var userID int64
	err := s.db.DB.Get(&userID, `INSERT INTO users (uid) VALUES ($1) RETURNING id`, "alice")
	assert.Nil(s.T(), err)

	// insert some things
	_, err = s.db.DB.Exec(`
		INSERT INTO things (uid, owner_id, serial_num, long, lat, location_identifier, last_sample)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())`, "1234", userID, "PA1", 12.2, 13.3, "LOC1",
	)
	assert.Nil(s.T(), err)

	// setup up simular mock for the thingful update request
	simular.ActivateNonDefault(s.client.Client)
	defer simular.DeactivateAndReset()

	simular.RegisterStubRequests(
		simular.NewStubRequest(
			"PATCH",
			"http://thingful.net/things/1234",
			simular.NewStringResponder(200, "{}"),
			simular.WithHeader(
				&http.Header{
					"Authorization": []string{"Bearer api-key"},
				},
			),
		),
	)

	ctx := logger.ToContext(context.Background(), s.logger)

	mux := goji.NewMux()
	handlers.RegisterLocationHandlers(mux, s.db, s.thingful)

	recorder := httptest.NewRecorder()

	input := []byte(`
	{
		"Code": "1234",
		"X": 31.2,
		"Y": 13.2
	}`)

	req, err := http.NewRequest(http.MethodPost, "/entity/locations/update", bytes.NewReader(input))
	assert.Nil(s.T(), err)
	req = req.WithContext(ctx)

	mux.ServeHTTP(recorder, req)
	assert.Equal(s.T(), http.StatusOK, recorder.Code)

	thing, err := s.db.GetThingByUID(ctx, "1234")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 31.2, thing.Longitude)
	assert.Equal(s.T(), 13.2, thing.Latitude)
}

func TestLocationHandlersSuite(t *testing.T) {
	suite.Run(t, new(LocationHandlersSuite))
}
