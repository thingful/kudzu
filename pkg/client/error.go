package client

// Error is a constant error type we use for sentinel errors
type Error string

// Error allows our custom error type to implement the error interface
func (e Error) Error() string { return string(e) }

const (
	// NotFoundError indicates a requested resource was not found
	NotFoundError = Error("Not found")

	// UnauthorizedError indicates we were unauthorized to fetch a requested
	// resource
	UnauthorizedError = Error("Unauthorized")

	// TimeoutError indicates the request timed out
	TimeoutError = Error("Timeout")

	// UnexpectedError is for all other HTTP errors
	UnexpectedError = Error("Unexpected")
)
