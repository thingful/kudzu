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
	"github.com/thingful/kuzu/pkg/logger"
	"github.com/thingful/kuzu/pkg/postgres"
)

type MetadataSuite struct {
	suite.Suite
	db     *postgres.DB
	logger kitlog.Logger
}

func (s *MetadataSuite) SetupTest() {
	logger := kitlog.NewNopLogger()
	connStr := os.Getenv("KUZU_DATABASE_URL")

	s.db = PrepareDB(s.T(), connStr, logger)
	s.logger = logger
}

func (s *MetadataSuite) TearDownTest() {
	CleanDB(s.T(), s.db)
}

func (s *MetadataSuite) TestGetMetadata() {
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

	metadata, err := s.db.GetMetadata(ctx)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), metadata, 7)
}

func TestMetadataSuite(t *testing.T) {
	suite.Run(t, new(MetadataSuite))
}
