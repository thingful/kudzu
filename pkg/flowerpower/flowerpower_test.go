package flowerpower_test

import (
	"io/ioutil"
	h "net/http"
	"testing"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/thingful/simular"

	"github.com/thingful/kuzu/pkg/flowerpower"
	"github.com/thingful/kuzu/pkg/http"
)

func TestSensorCount(t *testing.T) {
	logger := kitlog.NewNopLogger()
	client := http.NewClient(1*time.Second, logger)

	simular.Activate()
	defer simular.DeactivateAndReset()

	statusBytes, err := ioutil.ReadFile("testdata/barnabas_status.json")
	assert.Nil(t, err)

	simular.RegisterStubRequests(
		simular.NewStubRequest(
			"GET",
			flowerpower.StatusURL,
			simular.NewBytesResponder(200, statusBytes),
			simular.WithHeader(
				&h.Header{
					"Authorization": []string{"Bearer foo"},
				},
			),
		),
	)

	count, err := flowerpower.SensorCount(client, "foo")
	assert.Nil(t, err)
	assert.Equal(t, 36, count)

	err = simular.AllStubsCalled()
	assert.Nil(t, err)
}

func TestSensorCount404(t *testing.T) {
	logger := kitlog.NewNopLogger()
	client := http.NewClient(1*time.Second, logger)

	simular.Activate()
	defer simular.DeactivateAndReset()

	simular.RegisterStubRequests(
		simular.NewStubRequest(
			"GET",
			flowerpower.StatusURL,
			simular.NewStringResponder(404, "not found"),
			simular.WithHeader(
				&h.Header{
					"Authorization": []string{"Bearer foo"},
				},
			),
		),
	)

	_, err := flowerpower.SensorCount(client, "foo")
	assert.NotNil(t, err)

	err = simular.AllStubsCalled()
	assert.Nil(t, err)
}

func TestSensorCountInvalidResponse(t *testing.T) {
	logger := kitlog.NewNopLogger()
	client := http.NewClient(1*time.Second, logger)

	simular.Activate()
	defer simular.DeactivateAndReset()

	simular.RegisterStubRequests(
		simular.NewStubRequest(
			"GET",
			flowerpower.StatusURL,
			simular.NewStringResponder(200, "{\"locations"),
			simular.WithHeader(
				&h.Header{
					"Authorization": []string{"Bearer foo"},
				},
			),
		),
	)

	_, err := flowerpower.SensorCount(client, "foo")
	assert.NotNil(t, err)

	err = simular.AllStubsCalled()
	assert.Nil(t, err)
}
