package errors

import (
	. "code.cloudfoundry.org/cli/cf/i18n"
)

type UnbindableServiceError struct {
}

func NewUnbindableServiceError() error {
	return &UnbindableServiceError{}
}

func (err *UnbindableServiceError) Error() string {
	return T("This service doesn't support creation of keys.")
}
