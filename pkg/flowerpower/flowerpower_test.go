package flowerpower_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/thingful/simular"

	"github.com/thingful/kuzu/pkg/client"
	"github.com/thingful/kuzu/pkg/flowerpower"
)

func TestGetUser(t *testing.T) {
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

	user, err := flowerpower.GetUser(cl, "foo")
	assert.Nil(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "kovacs1barnabas1@gmail.com", user.ParrotID)

	err = simular.AllStubsCalled()
	assert.Nil(t, err)
}

func TestGetUserWhenInvalidToken(t *testing.T) {
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

	_, err := flowerpower.GetUser(cl, "foo")
	assert.NotNil(t, err)

	err = simular.AllStubsCalled()
	assert.Nil(t, err)
}

func TestGetLocations(t *testing.T) {
	logger := kitlog.NewNopLogger()
	cl := client.NewClient(1, logger)

	simular.Activate()
	defer simular.DeactivateAndReset()

	statusBytes, err := ioutil.ReadFile("testdata/barnabas_status.json")
	assert.Nil(t, err)

	configurationBytes, err := ioutil.ReadFile("testdata/barnabas_configuration.json")
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
		simular.NewStubRequest(
			"GET",
			flowerpower.ConfigurationURL,
			simular.NewBytesResponder(200, configurationBytes),
			simular.WithHeader(
				&http.Header{
					"Authorization": []string{"Bearer foo"},
				},
			),
		),
	)

	locations, err := flowerpower.GetLocations(cl, "foo")
	assert.Nil(t, err)
	assert.Len(t, locations, 35)

	assert.NotEqual(t, "", locations[0].SerialNum)

	err = simular.AllStubsCalled()
	assert.Nil(t, err)
}

func TestGetLocations404(t *testing.T) {
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

	_, err := flowerpower.GetLocations(cl, "foo")
	assert.NotNil(t, err)

	err = simular.AllStubsCalled()
	assert.Nil(t, err)
}

func TestGetLocationsInvalidResponse(t *testing.T) {
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

	_, err := flowerpower.GetLocations(cl, "foo")
	assert.NotNil(t, err)

	err = simular.AllStubsCalled()
	assert.Nil(t, err)
}

func TestGetReadingsOK(t *testing.T) {
	logger := kitlog.NewNopLogger()
	cl := client.NewClient(1, logger)

	simular.Activate()
	defer simular.DeactivateAndReset()

	dataBytes, err := ioutil.ReadFile("testdata/barnabas_data.json")
	assert.Nil(t, err)

	locationID := "Gu80jTmwyq1539530459586"
	fromTS := "2018-11-09T15:43:00Z"
	toTS := "2018-11-09T16:30:00Z"

	locationURL, _ := url.Parse(fmt.Sprintf(flowerpower.DataURL, locationID))
	q := locationURL.Query()
	q.Set("from_datetime_utc", fromTS)
	q.Set("to_datetime_utc", toTS)
	locationURL.RawQuery = q.Encode()

	simular.RegisterStubRequests(
		simular.NewStubRequest(
			"GET",
			locationURL.String(),
			simular.NewBytesResponder(200, dataBytes),
			simular.WithHeader(
				&http.Header{
					"Authorization": []string{"Bearer foo"},
				},
			),
		),
	)

	from, _ := time.Parse(time.RFC3339, fromTS)
	to, _ := time.Parse(time.RFC3339, toTS)

	readings, err := flowerpower.GetReadings(cl, "foo", locationID, from, to)
	assert.Nil(t, err)

	assert.Len(t, readings, 3)

	err = simular.AllStubsCalled()
	assert.Nil(t, err)
}
