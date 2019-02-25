package postgres_test

import (
	"context"
	"os"
	"testing"

	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/thingful/kudzu/pkg/logger"
	"github.com/thingful/kudzu/pkg/postgres"
	"github.com/thingful/kudzu/pkg/postgres/helper"
)

type DataSourcesSuite struct {
	suite.Suite
	db     *postgres.DB
	logger kitlog.Logger
}

func (s *DataSourcesSuite) SetupTest() {
	logger := kitlog.NewNopLogger()
	connStr := os.Getenv("KUDZU_DATABASE_URL")

	s.db = helper.PrepareDB(s.T(), connStr, logger)
	s.logger = logger
}

func (s *DataSourcesSuite) TearDownTest() {
	helper.CleanDB(s.T(), s.db)
}

func (s *DataSourcesSuite) TestGetDataSources() {
	_, err := s.db.DB.Exec(
		`INSERT INTO data_sources (name, unit, data_type) VALUES ($1, $2, $3)`,
		"air_temperature", "m3-lite:DegreeCelsius", "xsd:double",
	)
	assert.Nil(s.T(), err)

	_, err = s.db.DB.Exec(
		`INSERT INTO data_sources (name, data_type) VALUES ($1, $2)`,
		"fertilizer_level", "xsd:double",
	)
	assert.Nil(s.T(), err)

	ctx := logger.ToContext(context.Background(), s.logger)
	datasources, err := s.db.GetDataSources(ctx)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), datasources, 2)

	assert.Equal(s.T(), "air_temperature", datasources[0].Code)
	assert.Equal(s.T(), "m3-lite:DegreeCelsius", datasources[0].Unit.String)
	assert.Equal(s.T(), "xsd:double", datasources[0].DataType)

	assert.Equal(s.T(), "fertilizer_level", datasources[1].Code)
	assert.False(s.T(), datasources[1].Unit.Valid)
	assert.Equal(s.T(), "xsd:double", datasources[1].DataType)
}

func TestDataSourcesSuite(t *testing.T) {
	suite.Run(t, new(DataSourcesSuite))
}
