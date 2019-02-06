package postgres_test

import (
	"os"
	"testing"

	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/suite"

	"github.com/thingful/kuzu/pkg/postgres"
)

type UsersSuite struct {
	suite.Suite
	db *postgres.DB
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

	s.db = postgres.NewDB(connStr, logger)

	err = s.db.Start()
	if err != nil {
		s.T().Fatalf("Failed to start db service: %v", err)
	}
}

func (s *UsersSuite) TearDownTest() {
	err := s.db.Stop()
	if err != nil {
		s.T().Fatalf("Failed to stop db service: %v", err)
	}
}

func (s *UsersSuite) TestSaveUser() {
	//err := s.db.SaveUser("foobar")
	//assert.Nil(s.T(), err)
}

func TestUsersSuite(t *testing.T) {
	suite.Run(t, new(UsersSuite))
}
