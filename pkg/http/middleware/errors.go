package middleware

import (
	"encoding/json"
	"net/http"
)

type httpError struct {
	Name    int    `json:"Name"`
	Message string `json:"Message"`
}

func invalidTokenError(w http.ResponseWriter, err error) {
	httpErr := &httpError{
		Name:    http.StatusUnauthorized,
		Message: err.Error(),
	}

	b, err := json.Marshal(httpErr)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write(b)
}

func authForbiddenError(w http.ResponseWriter, err error) {
	httpErr := &httpError{
		Name:    http.StatusForbidden,
		Message: err.Error(),
	}

	b, err := json.Marshal(httpErr)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	w.Write(b)
}

func tooManyRequestsError(w http.ResponseWriter, err error) {
	httpErr := &httpError{
		Name:    http.StatusTooManyRequests,
		Message: err.Error(),
	}

	b, err := json.Marshal(httpErr)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	w.Write(b)
}
