package errors

import "fmt"

// NonRetryError is a type that noaa uses when it encountered an error,
// and is not going to retry the operation.  When errors of this type
// are encountered, they should result in a closed connection.
type NonRetryError struct {
	Err error
}

// NewNonRetryError constructs a NonRetryError from any error.
func NewNonRetryError(err error) NonRetryError {
	return NonRetryError{
		Err: err,
	}
}

// Error implements error.
func (e NonRetryError) Error() string {
	return fmt.Sprintf("Please ask your Cloud Foundry Operator to check the platform configuration: %s", e.Err.Error())
}
