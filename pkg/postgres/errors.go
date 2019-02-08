package postgres

const (
	// ConstraintError is the error code pq reports for a constraint violation
	// (null or empty string)
	ConstraintError = "23514"

	// UniqueViolationError is the error code pq reports for a unique index
	// violation
	UniqueViolationError = "23505"
)

// Error is a custom error type used for sentinel values
type Error string

// Error is the implementation of the error interface
func (e Error) Error() string { return string(e) }

const (
	// ClientError used to signal a client error, i.e. a duplicate value or missing
	// required field
	ClientError = Error("client error, i.e. non-unique violation or missing required field")

	// ServerError used to signal a server error that the client cannot fix
	ServerError = Error("server error - unexpected database error")
)
