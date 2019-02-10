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

type UsersSuite struct {
	suite.Suite
	db     *postgres.DB
	logger kitlog.Logger
}

func (s *UsersSuite) SetupTest() {
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

func (s *UsersSuite) TearDownTest() {
	err := postgres.Truncate(s.db.DB)
	if err != nil {
		s.T().Fatalf("Failed to truncate tables: %v", err)
	}

	err = s.db.Stop()
	if err != nil {
		s.T().Fatalf("Failed to stop db service: %v", err)
	}
}

func (s *UsersSuite) TestSaveUser() {
	ctx := logger.ToContext(context.Background(), s.logger)

	err := s.db.SaveUser(ctx, &postgres.User{
		UID:          "abc123",
		ParrotID:     "foo@example.com",
		AccessToken:  "access",
		RefreshToken: "refresh",
		Provider:     "parrot",
	})

	assert.Nil(s.T(), err)
	//assert.NotEqual(s.T(), int64(0), userID)

	//// ugly check we wrote the data ok
	//sql := `SELECT u.id, u.uid, u.parrot_id, i.access_token, i.refresh_token
	//FROM users u
	//JOIN identities i ON i.owner_id = u.id
	//WHERE u.id = $1`

	//var u postgres.User

	//err = s.db.DB.Get(&u, sql, userID)
	//assert.Nil(s.T(), err)

	//assert.Equal(s.T(), "abc123", u.UID)
	//assert.Equal(s.T(), "foo@example.com", u.ParrotID)
	//assert.Equal(s.T(), "access", u.AccessToken)
	//assert.Equal(s.T(), "refresh", u.RefreshToken)
}

func TestUsersSuite(t *testing.T) {
	suite.Run(t, new(UsersSuite))
}
