package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"goji.io"
	"goji.io/pat"

	"github.com/stretchr/testify/assert"
	"github.com/thingful/kudzu/pkg/http/middleware"
)

func TestRequestIDMiddleware(t *testing.T) {
	mux := goji.NewMux()
	mux.Use(middleware.RequestIDMiddleware)
	mux.Handle(pat.Get("/"), testHandler{})

	req, err := http.NewRequest(http.MethodGet, "/", nil)
	assert.Nil(t, err)

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	// assert that the middleware adds a request ID
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.NotEqual(t, "", recorder.Header().Get("X-Correlation-ID"))

	req.Header.Set("X-Correlation-ID", "foobar")

	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	// assert that the middleware uses a request ID I send
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "foobar", recorder.Header().Get("X-Correlation-ID"))
}
