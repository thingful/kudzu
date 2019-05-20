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
	"github.com/thingful/kudzu/pkg/http/handlers"
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

	mux := goji.NewMux()
	handlers.RegisterAppHandlers(mux, s.db)

	recorder := httptest.NewRecorder()
	input := []byte(`
	{
		"App": {
			"Name": "Student App"
		}
	}`)

	req, err := http.NewRequest(http.MethodPost, "/apps/new", bytes.NewReader(input))
	assert.Nil(s.T(), err)

	req = req.WithContext(ctx)

	mux.ServeHTTP(recorder, req)
	assert.Equal(s.T(), http.StatusCreated, recorder.Code)

	var resp map[string]interface{}

	err = json.Unmarshal(recorder.Body.Bytes(), &resp)
	assert.Nil(s.T(), err)

	assert.NotEqual(s.T(), "", resp["ApiKey"])
}

func TestAppsSuite(t *testing.T) {
	suite.Run(t, new(AppsSuite))
}
