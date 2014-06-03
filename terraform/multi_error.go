package terraform

import (
	"fmt"
	"strings"
)

// MultiError is an error type to track multiple errors. This is used to
// accumulate errors in cases such as configuration parsing, and returning
// them as a single error.
type MultiError struct {
	Errors []error
}

func (e *MultiError) Error() string {
	points := make([]string, len(e.Errors))
	for i, err := range e.Errors {
		points[i] = fmt.Sprintf("* %s", err)
	}

	return fmt.Sprintf(
		"%d error(s) occurred:\n\n%s",
		len(e.Errors), strings.Join(points, "\n"))
}

// MultiErrorAppend is a helper function that will append more errors
// onto a MultiError in order to create a larger multi-error. If the
// original error is not a MultiError, it will be turned into one.
func MultiErrorAppend(err error, errs ...error) *MultiError {
	if err == nil {
		err = new(MultiError)
	}

	switch err := err.(type) {
	case *MultiError:
		if err == nil {
			err = new(MultiError)
		}

		err.Errors = append(err.Errors, errs...)
		return err
	default:
		newErrs := make([]error, len(errs)+1)
		newErrs[0] = err
		copy(newErrs[1:], errs)
		return &MultiError{
			Errors: newErrs,
		}
	}
}
