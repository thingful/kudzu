package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/thingful/kudzu/pkg/http/handlers"
	"github.com/thingful/kudzu/pkg/http/middleware"
	"github.com/thingful/kudzu/pkg/logger"
	"github.com/thingful/kudzu/pkg/postgres"
	"github.com/thingful/kudzu/pkg/postgres/helper"
	goji "goji.io"
)

type AppsSuite struct {
	suite.Suite
	db     *postgres.DB
	logger kitlog.Logger
}

func (s *AppsSuite) SetupTest() {
	logger := kitlog.NewNopLogger()
	connStr := os.Getenv("KUDZU_DATABASE_URL")

	s.logger = logger
	s.db = helper.PrepareDB(s.T(), connStr, logger)
}

func (s *AppsSuite) TearDownTest() {
	helper.CleanDB(s.T(), s.db)
}

func (s *AppsSuite) TestCreateApp() {
	ctx := logger.ToContext(context.Background(), s.logger)

	// create an app with create users permission, and capture so we have an api
	// key we can use
	app, err := s.db.CreateApp(ctx, "Supervisor", postgres.ScopeClaims{postgres.CreateUserScope})
	assert.Nil(s.T(), err)

	// set up our mux with the create app handler for testing
	mux := goji.NewMux()
	handlers.RegisterAppHandlers(mux, s.db)

	// wrap the mux with our authentication middleware so that it will set up the
	// context correctly
	authMiddleware := middleware.NewAuthMiddleware(s.db)
	mux.Use(authMiddleware.Handler)

	// set up new test recorder that captures the response
	recorder := httptest.NewRecorder()
	input := []byte(`
	{
		"App": {
			"Name": "Student App"
		}
	}`)

	// create a new request to create a new app including our app api key
	req, err := http.NewRequest(http.MethodPost, "/apps/new", bytes.NewReader(input))
	assert.Nil(s.T(), err)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", app.Key))

	// add in our context for the quiet logger
	req = req.WithContext(ctx)

	mux.ServeHTTP(recorder, req)
	assert.Equal(s.T(), http.StatusCreated, recorder.Code)

	var resp map[string]interface{}

	err = json.Unmarshal(recorder.Body.Bytes(), &resp)
	assert.Nil(s.T(), err)

	assert.NotEqual(s.T(), "", resp["ApiKey"])
}

func (s *AppsSuite) TestCreateAppInvalidPermissions() {
	ctx := logger.ToContext(context.Background(), s.logger)

	// create an app with create users permission, and capture so we have an api
	// key we can use
	app, err := s.db.CreateApp(ctx, "Supervisor", postgres.ScopeClaims{postgres.GetTimeSeriesDataScope})
	assert.Nil(s.T(), err)

	// set up our mux with the create app handler for testing
	mux := goji.NewMux()
	handlers.RegisterAppHandlers(mux, s.db)

	// wrap the mux with our authentication middleware so that it will set up the
	// context correctly
	authMiddleware := middleware.NewAuthMiddleware(s.db)
	mux.Use(authMiddleware.Handler)

	// set up new test recorder that captures the response
	recorder := httptest.NewRecorder()
	input := []byte(`
	{
		"App": {
			"Name": "Student App"
		}
	}`)

	// create a new request to create a new app including our app api key
	req, err := http.NewRequest(http.MethodPost, "/apps/new", bytes.NewReader(input))
	assert.Nil(s.T(), err)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", app.Key))

	// add in our context for the quiet logger
	req = req.WithContext(ctx)

	mux.ServeHTTP(recorder, req)
	assert.Equal(s.T(), http.StatusForbidden, recorder.Code)
}

func (s *AppsSuite) TestCreateAppInvalidBody() {
	ctx := logger.ToContext(context.Background(), s.logger)

	// create an app with create users permission, and capture so we have an api
	// key we can use
	app, err := s.db.CreateApp(ctx, "Supervisor", postgres.ScopeClaims{postgres.CreateUserScope})
	assert.Nil(s.T(), err)

	// set up our mux with the create app handler for testing
	mux := goji.NewMux()
	handlers.RegisterAppHandlers(mux, s.db)

	// wrap the mux with our authentication middleware so that it will set up the
	// context correctly
	authMiddleware := middleware.NewAuthMiddleware(s.db)
	mux.Use(authMiddleware.Handler)

	testcases := []struct {
		label        string
		input        []byte
		expectedCode int
	}{
		{
			"empty name",
			[]byte(`{"App":{"Name":""}}`),
			http.StatusUnprocessableEntity,
		},
		{
			"invalid json",
			[]byte(`{"App":{"Name":""`),
			http.StatusUnprocessableEntity,
		},
	}

	for _, tc := range testcases {
		s.T().Run(tc.label, func(t *testing.T) {
			// set up new test recorder that captures the response
			recorder := httptest.NewRecorder()

			// create a new request to create a new app including our app api key
			req, err := http.NewRequest(http.MethodPost, "/apps/new", bytes.NewReader(tc.input))
			assert.Nil(s.T(), err)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", app.Key))

			// add in our context for the quiet logger
			req = req.WithContext(ctx)

			mux.ServeHTTP(recorder, req)
			assert.Equal(s.T(), tc.expectedCode, recorder.Code)
		})
	}
}

func TestAppsSuite(t *testing.T) {
	suite.Run(t, new(AppsSuite))
}
