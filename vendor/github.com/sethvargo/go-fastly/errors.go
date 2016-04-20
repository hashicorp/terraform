package fastly

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
)

// ErrMissingService is an error that is returned when an input struct requires
// a "Service" key, but one was not set.
var ErrMissingService = errors.New("Missing required field 'Service'")

// ErrMissingVersion is an error that is returned when an input struct requires
// a "Version" key, but one was not set.
var ErrMissingVersion = errors.New("Missing required field 'Version'")

// ErrMissingName is an error that is returned when an input struct requires a
// "Name" key, but one was not set.
var ErrMissingName = errors.New("Missing required field 'Name'")

// ErrMissingKey is an error that is returned when an input struct requires a
// "Name" key, but one was not set.
var ErrMissingKey = errors.New("Missing required field 'Key'")

// ErrMissingURL is an error that is returned when an input struct requires a
// "Name" key, but one was not set.
var ErrMissingURL = errors.New("Missing required field 'URL'")

// ErrMissingID is an error that is returned when an input struct requires an
// "ID" key, but one was not set.
var ErrMissingID = errors.New("Missing required field 'ID'")

// ErrMissingDictionary is an error that is returned when an input struct
// requires a "Dictionary" key, but one was not set.
var ErrMissingDictionary = errors.New("Missing required field 'Dictionary'")

// ErrMissingItemKey is an error that is returned when an input struct
// requires a "ItemKey" key, but one was not set.
var ErrMissingItemKey = errors.New("Missing required field 'ItemKey'")

// ErrMissingFrom is an error that is returned when an input struct
// requires a "From" key, but one was not set.
var ErrMissingFrom = errors.New("Missing required field 'From'")

// ErrMissingTo is an error that is returned when an input struct
// requires a "To" key, but one was not set.
var ErrMissingTo = errors.New("Missing required field 'To'")

// ErrMissingDirector is an error that is returned when an input struct
// requires a "From" key, but one was not set.
var ErrMissingDirector = errors.New("Missing required field 'Director'")

// ErrMissingBackend is an error that is returned when an input struct
// requires a "Backend" key, but one was not set.
var ErrMissingBackend = errors.New("Missing required field 'Backend'")

// ErrMissingYear is an error that is returned when an input struct
// requires a "Year" key, but one was not set.
var ErrMissingYear = errors.New("Missing required field 'Year'")

// ErrMissingMonth is an error that is returned when an input struct
// requires a "Month" key, but one was not set.
var ErrMissingMonth = errors.New("Missing required field 'Month'")

// Ensure HTTPError is, in fact, an error.
var _ error = (*HTTPError)(nil)

// HTTPError is a custom error type that wraps an HTTP status code with some
// helper functions.
type HTTPError struct {
	// StatusCode is the HTTP status code (2xx-5xx).
	StatusCode int

	// Message and Detail are information returned by the Fastly API.
	Message string `mapstructure:"msg"`
	Detail  string `mapstructure:"detail"`
}

// NewHTTPError creates a new HTTP error from the given code.
func NewHTTPError(resp *http.Response) *HTTPError {
	var e *HTTPError
	if resp.Body != nil {
		decodeJSON(&e, resp.Body)
	}
	e.StatusCode = resp.StatusCode
	return e
}

// Error implements the error interface and returns the string representing the
// error text that includes the status code and the corresponding status text.
func (e *HTTPError) Error() string {
	var r bytes.Buffer
	fmt.Fprintf(&r, "%d - %s", e.StatusCode, http.StatusText(e.StatusCode))

	if e.Message != "" {
		fmt.Fprintf(&r, "\nMessage: %s", e.Message)
	}

	if e.Detail != "" {
		fmt.Fprintf(&r, "\nDetail: %s", e.Detail)
	}

	return r.String()
}

// String implements the stringer interface and returns the string representing
// the string text that includes the status code and corresponding status text.
func (e *HTTPError) String() string {
	return e.Error()
}

// IsNotFound returns true if the HTTP error code is a 404, false otherwise.
func (e *HTTPError) IsNotFound() bool {
	return e.StatusCode == 404
}
