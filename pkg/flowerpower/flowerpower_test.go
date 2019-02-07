package flowerpower_test

import (
	"io/ioutil"
	"net/http"
	"testing"

	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/thingful/simular"

	"github.com/thingful/kuzu/pkg/client"
	"github.com/thingful/kuzu/pkg/flowerpower"
)

func TestUserExists(t *testing.T) {
	logger := kitlog.NewNopLogger()
	cl := client.NewClient(1, logger)

	simular.Activate()
	defer simular.DeactivateAndReset()

	profileBytes, err := ioutil.ReadFile("testdata/barnabas_profile.json")
	assert.Nil(t, err)

	simular.RegisterStubRequests(
		simular.NewStubRequest(
			"GET",
			flowerpower.ProfileURL,
			simular.NewBytesResponder(200, profileBytes),
			simular.WithHeader(
				&http.Header{
					"Authorization": []string{"Bearer foo"},
				},
			),
		),
	)

	exists := flowerpower.UserExists(cl, "foo")
	assert.True(t, exists)

	err = simular.AllStubsCalled()
	assert.Nil(t, err)
}

func TestUserExistsWhenInvalidToken(t *testing.T) {
	logger := kitlog.NewNopLogger()
	cl := client.NewClient(1, logger)

	simular.Activate()
	defer simular.DeactivateAndReset()

	simular.RegisterStubRequests(
		simular.NewStubRequest(
			"GET",
			flowerpower.ProfileURL,
			simular.NewStringResponder(401, "Unauthorized"),
			simular.WithHeader(
				&http.Header{
					"Authorization": []string{"Bearer foo"},
				},
			),
		),
	)

	exists := flowerpower.UserExists(cl, "foo")
	assert.False(t, exists)

	err := simular.AllStubsCalled()
	assert.Nil(t, err)
}

func TestSensorCount(t *testing.T) {
	logger := kitlog.NewNopLogger()
	cl := client.NewClient(1, logger)

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
				&http.Header{
					"Authorization": []string{"Bearer foo"},
				},
			),
		),
	)

	count, err := flowerpower.SensorCount(cl, "foo")
	assert.Nil(t, err)
	assert.Equal(t, 36, count)

	err = simular.AllStubsCalled()
	assert.Nil(t, err)
}

func TestSensorCount404(t *testing.T) {
	logger := kitlog.NewNopLogger()
	cl := client.NewClient(1, logger)

	simular.Activate()
	defer simular.DeactivateAndReset()

	simular.RegisterStubRequests(
		simular.NewStubRequest(
			"GET",
			flowerpower.StatusURL,
			simular.NewStringResponder(404, "not found"),
			simular.WithHeader(
				&http.Header{
					"Authorization": []string{"Bearer foo"},
				},
			),
		),
	)

	_, err := flowerpower.SensorCount(cl, "foo")
	assert.NotNil(t, err)

	err = simular.AllStubsCalled()
	assert.Nil(t, err)
}

func TestSensorCountInvalidResponse(t *testing.T) {
	logger := kitlog.NewNopLogger()
	cl := client.NewClient(1, logger)

	simular.Activate()
	defer simular.DeactivateAndReset()

	simular.RegisterStubRequests(
		simular.NewStubRequest(
			"GET",
			flowerpower.StatusURL,
			simular.NewStringResponder(200, "{\"locations"),
			simular.WithHeader(
				&http.Header{
					"Authorization": []string{"Bearer foo"},
				},
			),
		),
	)

	_, err := flowerpower.SensorCount(cl, "foo")
	assert.NotNil(t, err)

	err = simular.AllStubsCalled()
	assert.Nil(t, err)
}
