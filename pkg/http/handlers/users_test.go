package handlers_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/thingful/simular"
	goji "goji.io"

	"github.com/thingful/kuzu/pkg/client"
	"github.com/thingful/kuzu/pkg/flowerpower"
	"github.com/thingful/kuzu/pkg/http/handlers"
	"github.com/thingful/kuzu/pkg/postgres"
)

type UsersSuite struct {
	suite.Suite
	db     *postgres.DB
	logger kitlog.Logger
	client *client.Client
}

func (s *UsersSuite) SetupTest() {
	logger := kitlog.NewNopLogger()
	connStr := os.Getenv("KUZU_DATABASE_URL")

	db, err := postgres.Open(connStr)
	if err != nil {
		s.T().Fatalf("Failed to open db connection")
	}

	postgres.MigrateDownAll(db.DB, logger)

	err = db.Close()
	if err != nil {
		s.T().Fatalf("Failed to close db connection: %v", err)
	}

	s.logger = logger
	s.db = postgres.NewDB(connStr, logger, true)
	s.client = client.NewClient(1, logger)

	err = s.db.Start()
	if err != nil {
		s.T().Fatalf("Failed to start db service: %v", err)
	}
}

func (s *UsersSuite) TearDownTest() {
	err := postgres.Truncate(s.db.DB)
	if err != nil {
		s.T().Fatalf("Failed to truncate tables: %v", err)
	}

	err = s.db.Stop()
	if err != nil {
		s.T().Fatalf("Failed to stop db service: %v", err)
	}
}

func (s *UsersSuite) TestCreateUser() {
	// register simular http mocks
	profileBytes, err := ioutil.ReadFile("../../flowerpower/testdata/barnabas_profile.json")
	assert.Nil(s.T(), err)

	statusBytes, err := ioutil.ReadFile("../../flowerpower/testdata/barnabas_status.json")
	assert.Nil(s.T(), err)

	configurationBytes, err := ioutil.ReadFile("../../flowerpower/testdata/barnabas_configuration.json")
	assert.Nil(s.T(), err)

	simular.Activate()
	defer simular.DeactivateAndReset()

	simular.RegisterStubRequests(
		simular.NewStubRequest(
			"GET",
			flowerpower.ProfileURL,
			simular.NewBytesResponder(200, profileBytes),
			simular.WithHeader(
				&http.Header{
					"Authorization": []string{"Bearer access"},
				},
			),
		),
		simular.NewStubRequest(
			"GET",
			flowerpower.StatusURL,
			simular.NewBytesResponder(200, statusBytes),
			simular.WithHeader(
				&http.Header{
					"Authorization": []string{"Bearer access"},
				},
			),
		),
		simular.NewStubRequest(
			"GET",
			flowerpower.ConfigurationURL,
			simular.NewBytesResponder(200, configurationBytes),
			simular.WithHeader(
				&http.Header{
					"Authorization": []string{"Bearer access"},
				},
			),
		),
	)

	// create mux and register the handler we want to test
	mux := goji.NewMux()
	handlers.RegisterUserHandlers(mux, s.db, s.client)

	recorder := httptest.NewRecorder()

	input := []byte(`
	{
		"User": {
			"Identifier": "barnabas",
			"Provider": "parrot",
			"AccessToken": "access",
			"RefreshToken": "refresh"
		}
	}`)

	expected := `{"User": "barnabas","TotalThings": 35}`

	req, err := http.NewRequest(http.MethodPost, "/user/new", bytes.NewReader(input))
	assert.Nil(s.T(), err)

	mux.ServeHTTP(recorder, req)
	assert.Equal(s.T(), http.StatusAccepted, recorder.Code)
	assert.JSONEq(s.T(), expected, recorder.Body.String())

	err = simular.AllStubsCalled()
	assert.Nil(s.T(), err)
}

func (s *UsersSuite) TestCreateUserWhenAlreadyRegistered() {
	// register simular http mocks
	profileBytes, err := ioutil.ReadFile("../../flowerpower/testdata/barnabas_profile.json")
	assert.Nil(s.T(), err)

	simular.Activate()
	defer simular.DeactivateAndReset()

	sql := `INSERT INTO users (uid, parrot_id) VALUES ($1, $2)`
	_, err = s.db.DB.Exec(sql, "barnabas", "barnabas@example.com")
	assert.Nil(s.T(), err)

	simular.RegisterStubRequests(
		simular.NewStubRequest(
			"GET",
			flowerpower.ProfileURL,
			simular.NewBytesResponder(200, profileBytes),
			simular.WithHeader(
				&http.Header{
					"Authorization": []string{"Bearer access"},
				},
			),
		),
	)

	// create mux and register the handler we want to test
	mux := goji.NewMux()
	handlers.RegisterUserHandlers(mux, s.db, s.client)

	recorder := httptest.NewRecorder()

	input := []byte(`
	{
		"User": {
			"Identifier": "luca",
			"Provider": "parrot",
			"AccessToken": "access",
			"RefreshToken": "refresh"
		}
	}`)

	expected := `{"Message": "failed to insert user: client error, i.e. non-unique violation or missing required field","Name": 422}`

	req, err := http.NewRequest(http.MethodPost, "/user/new", bytes.NewReader(input))
	assert.Nil(s.T(), err)

	mux.ServeHTTP(recorder, req)
	assert.Equal(s.T(), http.StatusUnprocessableEntity, recorder.Code)
	assert.JSONEq(s.T(), expected, recorder.Body.String())

	err = simular.AllStubsCalled()
	assert.Nil(s.T(), err)
}

func TestUsersSuite(t *testing.T) {
	suite.Run(t, new(UsersSuite))
}
