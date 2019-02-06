package client_test

import (
	"fmt"
	h "net/http"
	"testing"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/thingful/simular"

	"github.com/thingful/kuzu/pkg/http"
	"github.com/thingful/kuzu/pkg/version"
)

func TestClient(t *testing.T) {
	logger := kitlog.NewNopLogger()

	client := http.NewClient(1*time.Second, logger)
	assert.NotNil(t, client)
}

func TestGet(t *testing.T) {
	logger := kitlog.NewNopLogger()
	client := http.NewClient(1*time.Second, logger)

	simular.Activate()
	defer simular.DeactivateAndReset()

	simular.RegisterStubRequests(
		simular.NewStubRequest(
			"GET",
			"http://example.com",
			simular.NewStringResponder(200, "ok"),
			simular.WithHeader(
				&h.Header{
					"Authorization": []string{"Bearer foo"},
					"User-Agent":    []string{fmt.Sprintf("grow(kuzu)/%s", version.Version)},
				},
			),
		),
	)

	b, err := client.Get("http://example.com", "foo")
	assert.Nil(t, err)
	assert.Equal(t, "ok", string(b))

	err = simular.AllStubsCalled()
	assert.Nil(t, err)
}

func TestGetNotFoundError(t *testing.T) {
	logger := kitlog.NewNopLogger()
	client := http.NewClient(1*time.Second, logger)

	simular.Activate()
	defer simular.DeactivateAndReset()

	simular.RegisterStubRequests(
		simular.NewStubRequest(
			"GET",
			"http://example.com",
			simular.NewStringResponder(404, "not found"),
			simular.WithHeader(
				&h.Header{
					"Authorization": []string{"Bearer foo"},
					"User-Agent":    []string{fmt.Sprintf("grow(kuzu)/%s", version.Version)},
				},
			),
		),
	)

	_, err := client.Get("http://example.com", "foo")
	assert.NotNil(t, err)

	err = simular.AllStubsCalled()
	assert.Nil(t, err)
}
