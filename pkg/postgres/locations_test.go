package postgres_test

import (
	"context"
	"os"
	"testing"

	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/thingful/kuzu/pkg/logger"
	"github.com/thingful/kuzu/pkg/postgres"
	"github.com/thingful/kuzu/pkg/postgres/helper"
)

type LocationsSuite struct {
	suite.Suite
	db     *postgres.DB
	logger kitlog.Logger
}

func (s *LocationsSuite) SetupTest() {
	logger := kitlog.NewNopLogger()
	connStr := os.Getenv("KUZU_DATABASE_URL")

	s.db = helper.PrepareDB(s.T(), connStr, logger)
	s.logger = logger
}

func (s *LocationsSuite) TearDownTest() {
	helper.CleanDB(s.T(), s.db)
}

func (s *LocationsSuite) TestListLocations() {
	var userID int64
	userUID := "abc123"

	err := s.db.DB.Get(&userID, `INSERT INTO users (uid) VALUES ($1) RETURNING id`, userUID)
	assert.Nil(s.T(), err)
	assert.NotEqual(s.T(), 0, userID)

	// insert some things
	_, err = s.db.DB.Exec(`
		INSERT INTO things (uid, owner_id, serial_num, long, lat, location_identifier, last_sample)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())`, "1234", userID, "PA1", 12.2, 13.3, "LOC1",
	)
	assert.Nil(s.T(), err)
	_, err = s.db.DB.Exec(`
		INSERT INTO things (uid, owner_id, serial_num, long, lat, location_identifier, last_sample)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())`, "1235", userID, "PA2", 0, 0, "LOC2",
	)
	assert.Nil(s.T(), err)
	_, err = s.db.DB.Exec(`
		INSERT INTO things (uid, owner_id, serial_num, long, lat, location_identifier, last_sample)
		VALUES ($1, $2, $3, $4, $5, $6, NOW() - interval '31 days')`, "1236", userID, "PA3", 12.2, 13.3, "LOC3",
	)
	assert.Nil(s.T(), err)

	ctx := logger.ToContext(context.Background(), s.logger)

	locations, err := s.db.ListLocations(ctx, "", false, false)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), locations, 3)

	locations, err = s.db.ListLocations(ctx, "abc123", false, false)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), locations, 3)

	locations, err = s.db.ListLocations(ctx, "foobar", false, false)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), locations, 0)

	locations, err = s.db.ListLocations(ctx, "", true, false)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), locations, 1)

	locations, err = s.db.ListLocations(ctx, "", false, true)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), locations, 1)
}

func (s *LocationsSuite) TestUpdateGeolocation() {
	var userID int64
	userUID := "abc123"

	err := s.db.DB.Get(&userID, `INSERT INTO users (uid) VALUES ($1) RETURNING id`, userUID)
	assert.Nil(s.T(), err)
	assert.NotEqual(s.T(), 0, userID)

	// insert some things
	_, err = s.db.DB.Exec(`
		INSERT INTO things (uid, owner_id, serial_num, long, lat, location_identifier, last_sample)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())`, "1234", userID, "PA1", 12.2, 13.3, "LOC1",
	)
	assert.Nil(s.T(), err)

	ctx := logger.ToContext(context.Background(), s.logger)

	loc, err := s.db.UpdateGeolocation(ctx, "1234", 25, 25)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), loc)

	locations, err := s.db.ListLocations(ctx, "", false, false)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), locations, 1)

	location := locations[0]
	assert.Equal(s.T(), 25.0, location.Longitude)
	assert.Equal(s.T(), 25.0, location.Latitude)
}

func TestLocationsSuite(t *testing.T) {
	suite.Run(t, new(LocationsSuite))
}
