package errors

import (
	. "code.cloudfoundry.org/cli/cf/i18n"
)

type NotAuthorizedError struct {
}

func NewNotAuthorizedError() error {
	return &NotAuthorizedError{}
}

func (err *NotAuthorizedError) Error() string {
	return T("Server error, status code: 403, error code: 10003, message: You are not authorized to perform the requested action")
}
