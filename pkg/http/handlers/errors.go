package handlers

import "encoding/json"

// Error is an interface for an error type we return from our custom handler
// type.
type Error interface {
	error
	Status() int
}

// HTTPError is our concrete implementation of the Error interface we return
// from handlers
type HTTPError struct {
	Code int
	Err  error
}

// Error returns the message
func (he *HTTPError) Error() string {
	return he.Err.Error()
}

// Status returns the status code associated with the error response.
func (he *HTTPError) Status() int {
	return he.Code
}

// MarshalJSON is our implementation of the json marshaller interface as we want
// to output the error message rather than the default error serialization which
// seems to be an empty json object.
func (he *HTTPError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Code    int    `json:"Name"`
		Message string `json:"Message"`
	}{
		Code:    he.Code,
		Message: he.Err.Error(),
	})
}
