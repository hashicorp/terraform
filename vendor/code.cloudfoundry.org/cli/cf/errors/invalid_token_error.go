package errors

import (
	. "code.cloudfoundry.org/cli/cf/i18n"
)

type InvalidTokenError struct {
	description string
}

func NewInvalidTokenError(description string) error {
	return &InvalidTokenError{description: description}
}

func (err *InvalidTokenError) Error() string {
	return T("Invalid auth token: ") + err.description
}
