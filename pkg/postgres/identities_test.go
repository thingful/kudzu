package postgres_test

import (
	"context"
	"os"

	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/thingful/kuzu/pkg/logger"
	"github.com/thingful/kuzu/pkg/postgres"
)

type IdentitiesSuite struct {
	suite.Suite
	db     *postgres.DB
	logger kitlog.Logger
}

func (s *IdentitiesSuite) SetupTest() {
	logger := kitlog.NewNopLogger()
	connStr := os.Getenv("KUZU_DATABASE_URL")

	db, err := postgres.Open(connStr)
	if err != nil {
		s.T().Fatalf("Failed to open db connection: %v", err)
	}

	postgres.MigrateDownAll(db.DB, logger)

	err = db.Close()
	if err != nil {
		s.T().Fatalf("Failed to close db connection: %v", err)
	}

	s.db = postgres.NewDB(connStr, logger, true)
	s.logger = logger

	err = s.db.Start()
	if err != nil {
		s.T().Fatalf("Failed to start db service: %v", err)
	}
}

func (s *IdentitiesSuite) TearDownTest() {
	err := postgres.Truncate(s.db.DB)
	if err != nil {
		s.T().Fatalf("Failed to truncate tables: %v", err)
	}

	err = s.db.Stop()
	if err != nil {
		s.T().Fatalf("Failed to stop db service: %v", err)
	}
}

func (s *IdentitiesSuite) TestNextAccessToken() {
	ctx := logger.ToContext(context.Background(), s.logger)

	var userID int64

	err := s.db.DB.Get(&userID, `INSERT INTO users (uid, parrot_id) VALUES ('abc123', 'bob@example.com') RETURNING id`)
	assert.Nil(s.T(), err)

	_, err = s.db.DB.Exec(`INSERT INTO identities (owner_id, auth_provider, access_token) VALUES ($1, 'parrot', $2)`, userID, "first")
	assert.Nil(s.T(), err)

	_, err = s.db.DB.Exec(`INSERT INTO identities (owner_id, auth_provider, access_token, indexed_at) VALUES ($1, 'parrot', $2, NOW() - interval '25 hours')`, userID, "second")
	assert.Nil(s.T(), err)

	_, err = s.db.DB.Exec(`INSERT INTO identities (owner_id, auth_provider, access_token, indexed_at) VALUES ($1, 'parrot', $2, NOW() - interval '1 hour')`, userID, "third")
	assert.Nil(s.T(), err)

	accessToken, err := s.db.NextAccessToken(ctx)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "first", accessToken)

	accessToken, err = s.db.NextAccessToken(ctx)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "second", accessToken)

	accessToken, err = s.db.NextAccessToken(ctx)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "", accessToken)

	accessToken, err = s.db.NextAccessToken(ctx)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "", accessToken)
}
