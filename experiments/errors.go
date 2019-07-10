package experiments

import (
	"fmt"
)

// UnavailableError is the error type returned by GetCurrent when the requested
// experiment is not recognized at all.
type UnavailableError struct {
	name string
}

func (e UnavailableError) Error() string {
	return fmt.Sprintf("no current experiment is named %q", e.name)
}

// DefunctError is the error type returned by GetCurrent when the requested
// experiment is recognized as defunct.
type DefunctError struct {
	msg string
}

func (e DefunctError) Error() string {
	return e.msg
}
