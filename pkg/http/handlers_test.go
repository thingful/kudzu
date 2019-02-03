package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthCheckHandler(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/pulse", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthCheckHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Unexpected response code, got %v, expected %v", status, http.StatusOK)
	}

	if rr.Body.String() != "ok" {
		t.Errorf("Unexpected response body, got %s, expected %s", rr.Body.String(), "ok")
	}
}
