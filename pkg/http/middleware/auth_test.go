package middleware_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/thingful/kudzu/pkg/http/middleware"
	"github.com/thingful/kudzu/pkg/postgres"
	"goji.io"
	"goji.io/pat"
)

type testHandler struct{}

func (th testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

type mockAppLoader struct {
	mock.Mock
}

func (m *mockAppLoader) LoadApp(ctx context.Context, token string) (*postgres.App, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(*postgres.App), args.Error(1)
}

func TestAuthMiddlewareOK(t *testing.T) {
	app := &postgres.App{
		UID:   "uid",
		Name:  "name",
		Roles: postgres.ScopeClaims{"timeseries"},
	}

	al := &mockAppLoader{}
	al.On("LoadApp", mock.Anything, "my-api-token").Return(app, nil)

	mux := goji.NewMux()
	auth := middleware.NewAuthMiddleware(al)

	mux.Handle(pat.Get("/"), testHandler{})
	mux.Use(auth.Handler)

	req, err := http.NewRequest(http.MethodGet, "/", nil)
	assert.Nil(t, err)

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", "my-api-token"))

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestAuthMiddlewareInvalidHeader(t *testing.T) {
	al := &mockAppLoader{}

	mux := goji.NewMux()
	auth := middleware.NewAuthMiddleware(al)

	mux.Handle(pat.Get("/"), testHandler{})
	mux.Use(auth.Handler)

	testcases := []struct {
		label  string
		header string
	}{
		{
			"missing header",
			"",
		},
		{
			"wrong scheme",
			"Token my-api-token",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.label, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "/", nil)
			assert.Nil(t, err)

			req.Header.Add("Authorization", tc.header)
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, req)

			assert.Equal(t, http.StatusUnauthorized, recorder.Code)
		})
	}

	//assert.JSONEq(t, `{"Message": "Invalid API Key", "Name": 403}`, recorder.Body.String())

}

func TestAuthMiddlewareUnknownToken(t *testing.T) {
	app := &postgres.App{
		UID:   "uid",
		Name:  "name",
		Roles: postgres.ScopeClaims{"timeseries"},
	}

	al := &mockAppLoader{}
	al.On("LoadApp", mock.Anything, "my-api-token").Return(app, errors.New("Invalid API Key"))

	mux := goji.NewMux()
	auth := middleware.NewAuthMiddleware(al)

	mux.Handle(pat.Get("/"), testHandler{})
	mux.Use(auth.Handler)

	req, err := http.NewRequest(http.MethodGet, "/", nil)
	assert.Nil(t, err)

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", "my-api-token"))

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.JSONEq(t, `{"Message": "Invalid API Key", "Name": 403}`, recorder.Body.String())

}
