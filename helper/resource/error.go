package resource

import (
	"fmt"
	"strings"
	"time"
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

	if e.Retries > 0 {
		return fmt.Sprintf("couldn't find resource (%d retries)", e.Retries)
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
	LastState     string
	Timeout       time.Duration
	ExpectedState []string
}

func (e *TimeoutError) Error() string {
	expectedState := "resource to be gone"
	if len(e.ExpectedState) > 0 {
		expectedState = fmt.Sprintf("state to become '%s'", strings.Join(e.ExpectedState, ", "))
	}

	extraInfo := make([]string, 0)
	if e.LastState != "" {
		extraInfo = append(extraInfo, fmt.Sprintf("last state: '%s'", e.LastState))
	}
	if e.Timeout > 0 {
		extraInfo = append(extraInfo, fmt.Sprintf("timeout: %s", e.Timeout.String()))
	}

	suffix := ""
	if len(extraInfo) > 0 {
		suffix = fmt.Sprintf(" (%s)", strings.Join(extraInfo, ", "))
	}

	if e.LastError != nil {
		return fmt.Sprintf("timeout while waiting for %s%s: %s",
			expectedState, suffix, e.LastError)
	}

	return fmt.Sprintf("timeout while waiting for %s%s",
		expectedState, suffix)
}
