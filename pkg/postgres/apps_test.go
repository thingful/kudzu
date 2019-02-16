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

func TestScopeClaimsValuer(t *testing.T) {
	sc := postgres.ScopeClaims{postgres.ScopeClaim("foo"), postgres.ScopeClaim("bar")}

	value, err := sc.Value()
	assert.Nil(t, err)

	assert.Equal(t, "{foo,bar}", value.(string))
}

type AppsSuite struct {
	suite.Suite
	db     *postgres.DB
	logger kitlog.Logger
}

func (s *AppsSuite) SetupTest() {
	logger := kitlog.NewNopLogger()
	connStr := os.Getenv("KUZU_DATABASE_URL")

	s.db = PrepareDB(s.T(), connStr, logger)
	s.logger = logger
}

func (s *AppsSuite) TearDownTest() {
	CleanDB(s.T(), s.db)
}

func (s *AppsSuite) TestCreateLoadApp() {
	ctx := logger.ToContext(context.Background(), s.logger)

	app, err := s.db.CreateApp(ctx, "app", []string{"create-users"})
	assert.Nil(s.T(), err)
	assert.NotEqual(s.T(), "", app.UID)
	assert.Equal(s.T(), "app", app.Name)
	assert.NotEqual(s.T(), "", app.Key)

	loadedApp, err := s.db.LoadApp(ctx, app.Key)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), loadedApp)
}

func (s *AppsSuite) TestInvalidClaim() {
	ctx := logger.ToContext(context.Background(), s.logger)

	_, err := s.db.CreateApp(ctx, "app", []string{"create-bananas"})
	assert.NotNil(s.T(), err)
}

func (s *AppsSuite) TestLoadMissingApp() {
	ctx := logger.ToContext(context.Background(), s.logger)

	_, err := s.db.LoadApp(ctx, "foo-bar")
	assert.NotNil(s.T(), err)
}

func (s *AppsSuite) TestLoadInvalidKey() {
	ctx := logger.ToContext(context.Background(), s.logger)

	_, err := s.db.LoadApp(ctx, "foo")
	assert.NotNil(s.T(), err)
}

func TestAppsSuite(t *testing.T) {
	suite.Run(t, new(AppsSuite))
}
