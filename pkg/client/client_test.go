package client_test

import (
	"fmt"
	"net/http"
	"testing"

	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/thingful/simular"

	"github.com/thingful/kuzu/pkg/client"
	"github.com/thingful/kuzu/pkg/version"
)

func TestClient(t *testing.T) {
	logger := kitlog.NewNopLogger()

	cl := client.NewClient(1, logger)
	assert.NotNil(t, cl)
}

func TestGet(t *testing.T) {
	logger := kitlog.NewNopLogger()
	cl := client.NewClient(1, logger)

	simular.Activate()
	defer simular.DeactivateAndReset()

	simular.RegisterStubRequests(
		simular.NewStubRequest(
			"GET",
			"http://example.com",
			simular.NewStringResponder(200, "ok"),
			simular.WithHeader(
				&http.Header{
					"Authorization": []string{"Bearer foo"},
					"User-Agent":    []string{fmt.Sprintf("grow(kuzu)/%s", version.Version)},
				},
			),
		),
	)

	b, err := cl.Get("http://example.com", "foo")
	assert.Nil(t, err)
	assert.Equal(t, "ok", string(b))

	err = simular.AllStubsCalled()
	assert.Nil(t, err)
}

func TestGetNotFoundError(t *testing.T) {
	logger := kitlog.NewNopLogger()
	cl := client.NewClient(1, logger)

	simular.Activate()
	defer simular.DeactivateAndReset()

	simular.RegisterStubRequests(
		simular.NewStubRequest(
			"GET",
			"http://example.com",
			simular.NewStringResponder(404, "not found"),
			simular.WithHeader(
				&http.Header{
					"Authorization": []string{"Bearer foo"},
					"User-Agent":    []string{fmt.Sprintf("grow(kuzu)/%s", version.Version)},
				},
			),
		),
	)

	_, err := cl.Get("http://example.com", "foo")
	assert.NotNil(t, err)

	err = simular.AllStubsCalled()
	assert.Nil(t, err)
}
