package client_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/thingful/simular"

	"github.com/thingful/kudzu/pkg/client"
	"github.com/thingful/kudzu/pkg/logger"
	"github.com/thingful/kudzu/pkg/version"
)

func TestClient(t *testing.T) {
	cl := client.NewClient(1, false)
	assert.NotNil(t, cl)
}

func TestGet(t *testing.T) {
	cl := client.NewClient(1, false)

	simular.ActivateNonDefault(cl.Client)
	defer simular.DeactivateAndReset()

	simular.RegisterStubRequests(
		simular.NewStubRequest(
			"GET",
			"http://example.com",
			simular.NewStringResponder(200, "ok"),
			simular.WithHeader(
				&http.Header{
					"Authorization": []string{"Bearer foo"},
					"User-Agent":    []string{fmt.Sprintf("grow(kudzu)/%s", version.Version)},
				},
			),
		),
	)

	b, err := cl.Get(context.Background(), "http://example.com", "foo")
	assert.Nil(t, err)
	assert.Equal(t, "ok", string(b))

	err = simular.AllStubsCalled()
	assert.Nil(t, err)
}

func TestGetNotFoundError(t *testing.T) {
	cl := client.NewClient(1, false)

	simular.ActivateNonDefault(cl.Client)
	defer simular.DeactivateAndReset()

	simular.RegisterStubRequests(
		simular.NewStubRequest(
			"GET",
			"http://example.com",
			simular.NewStringResponder(404, "not found"),
			simular.WithHeader(
				&http.Header{
					"Authorization": []string{"Bearer foo"},
					"User-Agent":    []string{fmt.Sprintf("grow(kudzu)/%s", version.Version)},
				},
			),
		),
	)

	log := kitlog.NewNopLogger()

	_, err := cl.Get(logger.ToContext(context.Background(), log), "http://example.com", "foo")
	assert.NotNil(t, err)

	err = simular.AllStubsCalled()
	assert.Nil(t, err)
}
