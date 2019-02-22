package handlers_test

import (
	"bytes"
	"context"
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
	"github.com/thingful/kuzu/pkg/indexer"
	"github.com/thingful/kuzu/pkg/logger"
	"github.com/thingful/kuzu/pkg/postgres"
	"github.com/thingful/kuzu/pkg/postgres/helper"
)

type UsersSuite struct {
	suite.Suite
	db      *postgres.DB
	logger  kitlog.Logger
	client  *client.Client
	indexer *indexer.Indexer
}

func (s *UsersSuite) SetupTest() {
	logger := kitlog.NewNopLogger()
	connStr := os.Getenv("KUZU_DATABASE_URL")

	s.logger = logger
	s.db = helper.PrepareDB(s.T(), connStr, logger)

	s.client = client.NewClient(1, true)
	s.indexer = indexer.NewIndexer(
		&indexer.Config{
			DB:     s.db,
			Client: s.client,
		}, logger)
}

func (s *UsersSuite) TearDownTest() {
	helper.CleanDB(s.T(), s.db)
}

func (s *UsersSuite) TestCreateUser() {
	ctx := logger.ToContext(context.Background(), s.logger)

	// register simular http mocks
	profileBytes, err := ioutil.ReadFile("../../flowerpower/testdata/barnabas_profile.json")
	assert.Nil(s.T(), err)

	statusBytes, err := ioutil.ReadFile("../../flowerpower/testdata/barnabas_status.json")
	assert.Nil(s.T(), err)

	configurationBytes, err := ioutil.ReadFile("../../flowerpower/testdata/barnabas_configuration.json")
	assert.Nil(s.T(), err)

	simular.ActivateNonDefault(s.client.Client)
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
	handlers.RegisterUserHandlers(mux, s.db, s.client, s.indexer)

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
	req = req.WithContext(ctx)
	assert.Nil(s.T(), err)

	mux.ServeHTTP(recorder, req)
	assert.Equal(s.T(), http.StatusAccepted, recorder.Code)
	assert.JSONEq(s.T(), expected, recorder.Body.String())

	err = simular.AllStubsCalled()
	assert.Nil(s.T(), err)
}

func (s *UsersSuite) TestCreateUserWhenAlreadyRegistered() {
	ctx := logger.ToContext(context.Background(), s.logger)

	// register simular http mocks
	profileBytes, err := ioutil.ReadFile("../../flowerpower/testdata/barnabas_profile.json")
	assert.Nil(s.T(), err)

	simular.ActivateNonDefault(s.client.Client)
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
	handlers.RegisterUserHandlers(mux, s.db, s.client, s.indexer)

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
	req = req.WithContext(ctx)

	mux.ServeHTTP(recorder, req)
	assert.Equal(s.T(), http.StatusUnprocessableEntity, recorder.Code)
	assert.JSONEq(s.T(), expected, recorder.Body.String())

	err = simular.AllStubsCalled()
	assert.Nil(s.T(), err)
}

func (s *UsersSuite) TestDeleteUser() {
	ctx := logger.ToContext(context.Background(), s.logger)

	sql := `INSERT INTO users (uid, parrot_id) VALUES ($1, $2)`
	_, err := s.db.DB.Exec(sql, "barnabas", "barnabas@example.com")
	assert.Nil(s.T(), err)

	mux := goji.NewMux()
	handlers.RegisterUserHandlers(mux, s.db, s.client, s.indexer)

	recorder := httptest.NewRecorder()
	input := []byte(`
	{
		"User": {
			"Identifier": "barnabas"
		}
	}`)

	req, err := http.NewRequest(http.MethodDelete, "/user/delete", bytes.NewReader(input))
	assert.Nil(s.T(), err)

	req = req.WithContext(ctx)
	mux.ServeHTTP(recorder, req)
	assert.Equal(s.T(), http.StatusNoContent, recorder.Code)
}

func TestUsersSuite(t *testing.T) {
	suite.Run(t, new(UsersSuite))
}
