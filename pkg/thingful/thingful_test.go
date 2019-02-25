package thingful_test

import (
	"context"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/guregu/null"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/thingful/kudzu/pkg/client"
	"github.com/thingful/kudzu/pkg/flowerpower"
	"github.com/thingful/kudzu/pkg/logger"
	"github.com/thingful/kudzu/pkg/postgres"
	"github.com/thingful/kudzu/pkg/thingful"
	"github.com/thingful/simular"
)

type ThingfulSuite struct {
	suite.Suite
	httpClient *client.Client
	thingful   *thingful.Thingful
	logger     kitlog.Logger
}

func (s *ThingfulSuite) SetupTest() {
	s.logger = kitlog.NewNopLogger()
	s.httpClient = client.NewClient(1, true)
	s.thingful = thingful.NewClient(s.httpClient, "https://thingful.net", "foobar", true, 2)
}

func (s *ThingfulSuite) TestCreateThing() {
	ctx := logger.ToContext(context.Background(), s.logger)

	createResponseBytes, err := ioutil.ReadFile("./testdata/create_response.json")
	assert.Nil(s.T(), err)

	simular.ActivateNonDefault(s.httpClient.Client)
	defer simular.DeactivateAndReset()

	simular.RegisterStubRequests(
		simular.NewStubRequest(
			"POST",
			"https://thingful.net/things",
			simular.NewBytesResponder(200, createResponseBytes),
			simular.WithHeader(
				&http.Header{
					"Authorization": []string{"Bearer foobar"},
				},
			),
		),
	)

	indexedAt, _ := time.Parse(time.RFC3339, "2019-02-23T23:28:47Z")
	lastSample, _ := time.Parse(time.RFC3339, "2019-02-23T23:28:47Z")

	postgresThing := &postgres.Thing{
		Nickname:      null.StringFrom("Plant 1"),
		IndexedAt:     null.TimeFrom(indexedAt),
		SerialNum:     "PA1234",
		Longitude:     -7.92494,
		Latitude:      54.98063,
		LastSampleUTC: null.TimeFrom(lastSample),
	}

	readingTs1, _ := time.Parse(time.RFC3339, "2018-10-20T13:04:08Z")
	readingTs2, _ := time.Parse(time.RFC3339, "2018-10-20T12:49:08Z")

	readings := []flowerpower.Reading{
		flowerpower.Reading{
			Timestamp:              readingTs1,
			CalibratedSoilMoisture: 41.24,
		},
		flowerpower.Reading{
			Timestamp:              readingTs2,
			CalibratedSoilMoisture: 41.24,
		},
	}

	uid, err := s.thingful.CreateThing(ctx, postgresThing, readings)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "ds6r2tvx", uid)
}

func TestThingfulClient(t *testing.T) {
	suite.Run(t, new(ThingfulSuite))
}
