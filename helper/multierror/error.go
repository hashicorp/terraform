package multierror

import (
	"fmt"
	"strings"
)

// Error is an error type to track multiple errors. This is used to
// accumulate errors in cases such as configuration parsing, and returning
// them as a single error.
type Error struct {
	Errors []error
}

func (e *Error) Error() string {
	points := make([]string, len(e.Errors))
	for i, err := range e.Errors {
		points[i] = fmt.Sprintf("* %s", err)
	}

	return fmt.Sprintf(
		"%d error(s) occurred:\n\n%s",
		len(e.Errors), strings.Join(points, "\n"))
}

func (e *Error) GoString() string {
	return fmt.Sprintf("*%#v", *e)
}

// ErrorAppend is a helper function that will append more errors
// onto an Error in order to create a larger multi-error. If the
// original error is not an Error, it will be turned into one.
func ErrorAppend(err error, errs ...error) *Error {
	if err == nil {
		err = new(Error)
	}

	switch err := err.(type) {
	case *Error:
		if err == nil {
			err = new(Error)
		}

		err.Errors = append(err.Errors, errs...)
		return err
	default:
		newErrs := make([]error, len(errs)+1)
		newErrs[0] = err
		copy(newErrs[1:], errs)
		return &Error{
			Errors: newErrs,
		}
	}
}
