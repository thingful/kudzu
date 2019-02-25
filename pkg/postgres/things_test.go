package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/guregu/null"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/thingful/kudzu/pkg/logger"
	"github.com/thingful/kudzu/pkg/postgres"
	"github.com/thingful/kudzu/pkg/postgres/helper"
)

type ThingsSuite struct {
	suite.Suite
	db     *postgres.DB
	logger kitlog.Logger
}

func (s *ThingsSuite) SetupTest() {
	logger := kitlog.NewNopLogger()
	connStr := os.Getenv("KUDZU_DATABASE_URL")

	s.db = helper.PrepareDB(s.T(), connStr, logger)
	s.logger = logger
}

func (s *ThingsSuite) TearDownTest() {
	helper.CleanDB(s.T(), s.db)
}

func (s *ThingsSuite) TestRoundTrip() {
	var userID int64
	userUID := "abc123"

	err := s.db.DB.Get(&userID, `INSERT INTO users (uid) VALUES ($1) RETURNING id`, userUID)
	assert.Nil(s.T(), err)
	assert.NotEqual(s.T(), 0, userID)

	ctx := logger.ToContext(context.Background(), s.logger)

	now := time.Now()

	thing := &postgres.Thing{
		UID:             null.StringFrom("abc123"),
		OwnerID:         userID,
		Provider:        null.StringFrom("parrot"),
		SerialNum:       "PA123",
		Longitude:       12.2,
		Latitude:        15.4,
		FirstSampleUTC:  null.TimeFrom(now),
		LastSampleUTC:   null.TimeFrom(now),
		CreatedAt:       null.TimeFrom(now),
		UpdatedAt:       null.TimeFrom(now),
		IndexedAt:       null.TimeFrom(now),
		LastUploadedUTC: null.TimeFrom(now),
		Nickname:        null.StringFrom("My Plant"),
		LocationID:      "abc123",
	}

	err = s.db.CreateThing(ctx, thing)
	assert.Nil(s.T(), err)

	readThing, err := s.db.GetThing(ctx, "abc123")
	assert.Nil(s.T(), err)

	assert.Equal(s.T(), "PA123", readThing.SerialNum)

	now = time.Now()

	readThing.IndexedAt = null.TimeFrom(now)
	readThing.LastUploadedUTC = null.TimeFrom(now)
	readThing.Nickname = null.StringFrom("New Plant")
	readThing.Longitude = 0
	readThing.Latitude = 0

	err = s.db.UpdateThing(ctx, readThing)
	assert.Nil(s.T(), err)

	// verify that the geolocation has not changed
	readThing, err = s.db.GetThing(ctx, "abc123")
	assert.Nil(s.T(), err)

	assert.Equal(s.T(), 12.2, readThing.Longitude)
	assert.Equal(s.T(), 15.4, readThing.Latitude)
}

func (s *ThingsSuite) TestGetUnknownThing() {
	ctx := logger.ToContext(context.Background(), s.logger)

	_, err := s.db.GetThing(ctx, "foobar")
	assert.NotNil(s.T(), err)
}

func TestThingsSuite(t *testing.T) {
	suite.Run(t, new(ThingsSuite))
}
