package postgres_test

import (
	"os"
	"testing"

	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/suite"

	"github.com/thingful/kuzu/pkg/postgres"
)

type LocationsSuite struct {
	suite.Suite
	db     *postgres.DB
	logger kitlog.Logger
}

func (s *LocationsSuite) SetupTest() {
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

func (s *LocationsSuite) TearDownTest() {
	err := postgres.Truncate(s.db.DB)
	if err != nil {
		s.T().Fatalf("Failed to truncate tables: %v", err)
	}

	err = s.db.Stop()
	if err != nil {
		s.T().Fatalf("Failed to stop db service: %v", err)
	}
}

//sfunc (s *LocationsSuite) TestSaveLocations() {
//s	firstTs := "2018-01-01T00:00:00Z"
//s	lastTs := "2018-01-02T00:00:00Z"
//s
//s	first, _ := time.Parse(time.RFC3339, firstTs)
//s	last, _ := time.Parse(time.RFC3339, lastTs)
//s
//s	assert.NotNil(s.T(), first)
//s	assert.NotNil(s.T(), last)
//s	locations := []flowerpower.Location{
//s		{
//s			Nickname:       "plant1",
//s			LocationID:     "abc123",
//s			SerialNum:      "PA123",
//s			FirstSampleUTC: first,
//s			LastSampleUTC:  last,
//s			Longitude:      12,
//s			Latitude:       12,
//s		},
//s		{
//s			Nickname:       "plant2",
//s			LocationID:     "abc124",
//s			SerialNum:      "PA124",
//s			FirstSampleUTC: first,
//s			LastSampleUTC:  last,
//s			Longitude:      12,
//s			Latitude:       12,
//s		},
//s		{
//s			Nickname:       "plant3",
//s			LocationID:     "abc125",
//s			SerialNum:      "PA124",
//s			FirstSampleUTC: first,
//s			LastSampleUTC:  last,
//s			Longitude:      12,
//s			Latitude:       12,
//s		},
//s	}
//s
//s	ctx := logger.ToContext(context.Background(), s.logger)
//s
//s	err := s.db.SaveUser(
//s		ctx,
//s		&postgres.User{
//s			UID:          "abc123",
//s			ParrotID:     "foo@example.com",
//s			AccessToken:  "accessToken",
//s			RefreshToken: "refreshToken",
//s			Provider:     "parrot",
//s		},
//s	)
//s	assert.Nil(s.T(), err)
//s
//s	//err = s.db.SaveLocations(
//s	//ctx,
//s	//userID,
//s	//locations,
//s	//)
//s	//assert.Nil(s.T(), err)
//s}

//func (s *LocationsSuite) TestSaveInvalidLocations() {
//	firstTs := "2018-01-01T00:00:00Z"
//	lastTs := "2018-01-02T00:00:00Z"
//
//	first, _ := time.Parse(time.RFC3339, firstTs)
//	last, _ := time.Parse(time.RFC3339, lastTs)
//
//	ctx := logger.ToContext(context.Background(), s.logger)
//
//	userID, err := s.db.SaveUser(
//		ctx,
//		&postgres.User{
//			UID:          "abc123",
//			ParrotID:     "foo@example.com",
//			AccessToken:  "accessToken",
//			RefreshToken: "refreshToken",
//			Provider:     "parrot",
//		},
//	)
//	assert.Nil(s.T(), err)
//
//	testcases := []struct {
//		label     string
//		locations []flowerpower.Location
//	}{
//		{
//			label: "duplicate location identifier",
//			locations: []flowerpower.Location{
//				{
//					Nickname:       "plant1",
//					LocationID:     "abc123",
//					SerialNum:      "PA123",
//					FirstSampleUTC: first,
//					LastSampleUTC:  last,
//					Longitude:      12,
//					Latitude:       12,
//				},
//				{
//					Nickname:       "plant2",
//					LocationID:     "abc123",
//					SerialNum:      "PA124",
//					FirstSampleUTC: first,
//					LastSampleUTC:  last,
//					Longitude:      12,
//					Latitude:       12,
//				},
//			},
//		},
//		{
//			label: "blank location identifier",
//			locations: []flowerpower.Location{
//				{
//					Nickname:       "plant1",
//					LocationID:     "",
//					SerialNum:      "PA123",
//					FirstSampleUTC: first,
//					LastSampleUTC:  last,
//					Longitude:      12,
//					Latitude:       12,
//				},
//				{
//					Nickname:       "plant2",
//					LocationID:     "abc123",
//					SerialNum:      "PA124",
//					FirstSampleUTC: first,
//					LastSampleUTC:  last,
//					Longitude:      12,
//					Latitude:       12,
//				},
//			},
//		},
//		{
//			label: "blank serial_num",
//			locations: []flowerpower.Location{
//				{
//					Nickname:       "plant1",
//					LocationID:     "abc123",
//					SerialNum:      "",
//					FirstSampleUTC: first,
//					LastSampleUTC:  last,
//					Longitude:      12,
//					Latitude:       12,
//				},
//				{
//					Nickname:       "plant2",
//					LocationID:     "abc124",
//					SerialNum:      "PA124",
//					FirstSampleUTC: first,
//					LastSampleUTC:  last,
//					Longitude:      12,
//					Latitude:       12,
//				},
//			},
//		},
//	}
//
//	for _, tc := range testcases {
//		s.T().Run(tc.label, func(t *testing.T) {
//			err := s.db.SaveLocations(
//				ctx,
//				userID,
//				tc.locations,
//			)
//			assert.NotNil(s.T(), err)
//		})
//	}
//}

func TestLocationsSuite(t *testing.T) {
	suite.Run(t, new(LocationsSuite))
}
