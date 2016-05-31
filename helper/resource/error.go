package resource

import (
	"fmt"
	"strings"
)

type NotFoundError struct {
	LastError    error
	LastRequest  interface{}
	LastResponse interface{}
	Message      string
	Retries      int
}

func (e *NotFoundError) Error() string {
	if e.Message != "" {
		return e.Message
	}

	return "couldn't find resource"
}

// UnexpectedStateError is returned when Refresh returns a state that's neither in Target nor Pending
type UnexpectedStateError struct {
	LastError     error
	State         string
	ExpectedState []string
}

func (e *UnexpectedStateError) Error() string {
	return fmt.Sprintf(
		"unexpected state '%s', wanted target '%s'. last error: %s",
		e.State,
		strings.Join(e.ExpectedState, ", "),
		e.LastError,
	)
}

// TimeoutError is returned when WaitForState times out
type TimeoutError struct {
	LastError     error
	ExpectedState []string
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf(
		"timeout while waiting for state to become '%s'. last error: %s",
		strings.Join(e.ExpectedState, ", "),
		e.LastError,
	)
}
