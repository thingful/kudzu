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
)

type IdentitiesSuite struct {
	suite.Suite
	db     *postgres.DB
	logger kitlog.Logger
}

func (s *IdentitiesSuite) SetupTest() {
	logger := kitlog.NewNopLogger()
	connStr := os.Getenv("KUZU_DATABASE_URL")

	s.db = PrepareDB(s.T(), connStr, logger)
	s.logger = logger
}

func (s *IdentitiesSuite) TearDownTest() {
	CleanDB(s.T(), s.db)
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

	identity, err := s.db.NextIdentity(ctx)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "first", identity.AccessToken)

	identity, err = s.db.NextIdentity(ctx)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "second", identity.AccessToken)

	identity, err = s.db.NextIdentity(ctx)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "", identity.AccessToken)

	identity, err = s.db.NextIdentity(ctx)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "", identity.AccessToken)
}

func TestIdentitiesSuite(t *testing.T) {
	suite.Run(t, new(IdentitiesSuite))
}
