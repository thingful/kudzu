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

type UsersSuite struct {
	suite.Suite
	db     *postgres.DB
	logger kitlog.Logger
}

func (s *UsersSuite) SetupTest() {
	logger := kitlog.NewNopLogger()
	connStr := os.Getenv("KUZU_DATABASE_URL")

	s.db = helper.PrepareDB(s.T(), connStr, logger)
	s.logger = logger
}

func (s *UsersSuite) TearDownTest() {
	helper.CleanDB(s.T(), s.db)
}

func (s *UsersSuite) TestSaveUser() {
	ctx := logger.ToContext(context.Background(), s.logger)

	userID, err := s.db.SaveUser(ctx, &postgres.User{
		UID:          "abc123",
		ParrotID:     "foo@example.com",
		AccessToken:  "access",
		RefreshToken: "refresh",
		Provider:     "parrot",
	})

	assert.Nil(s.T(), err)
	assert.NotEqual(s.T(), int64(0), userID)

	// ugly check we wrote the data ok
	sql := `SELECT u.id, u.uid, u.parrot_id, i.access_token, i.refresh_token
	FROM users u
	JOIN identities i ON i.owner_id = u.id
	WHERE u.id = $1`

	var u postgres.User

	err = s.db.DB.Get(&u, sql, userID)
	assert.Nil(s.T(), err)

	assert.Equal(s.T(), "abc123", u.UID)
	assert.Equal(s.T(), "foo@example.com", u.ParrotID)
	assert.Equal(s.T(), "access", u.AccessToken)
	assert.Equal(s.T(), "refresh", u.RefreshToken)

	err = s.db.DeleteUser(ctx, "abc123")
	assert.Nil(s.T(), err)

	err = s.db.DB.Get(&u, sql, userID)
	assert.NotNil(s.T(), err)
}

func (s *UsersSuite) TestDeleteUnknownUser() {
	ctx := logger.ToContext(context.Background(), s.logger)

	err := s.db.DeleteUser(ctx, "abc123")
	assert.Nil(s.T(), err)
}

func TestUsersSuite(t *testing.T) {
	suite.Run(t, new(UsersSuite))
}
