package errors

import (
	"fmt"

	. "code.cloudfoundry.org/cli/cf/i18n"
)

//go:generate counterfeiter . HTTPError

type HTTPError interface {
	Error() string
	StatusCode() int   // actual HTTP status code
	ErrorCode() string // error code returned in response body from CC or UAA
}

type baseHTTPError struct {
	statusCode   int
	apiErrorCode string
	description  string
}

type HTTPNotFoundError struct {
	baseHTTPError
}

func NewHTTPError(statusCode int, code string, description string) error {
	err := baseHTTPError{
		statusCode:   statusCode,
		apiErrorCode: code,
		description:  description,
	}
	switch statusCode {
	case 404:
		return &HTTPNotFoundError{err}
	default:
		return &err
	}
}

func (err *baseHTTPError) StatusCode() int {
	return err.statusCode
}

func (err *baseHTTPError) Error() string {
	return fmt.Sprintf(T("Server error, status code: {{.ErrStatusCode}}, error code: {{.ErrAPIErrorCode}}, message: {{.ErrDescription}}",
		map[string]interface{}{"ErrStatusCode": err.statusCode,
			"ErrAPIErrorCode": err.apiErrorCode,
			"ErrDescription":  err.description}),
	)
}

func (err *baseHTTPError) ErrorCode() string {
	return err.apiErrorCode
}
