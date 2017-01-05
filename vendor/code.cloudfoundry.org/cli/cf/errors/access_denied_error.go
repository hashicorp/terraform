package errors

import . "code.cloudfoundry.org/cli/cf/i18n"

type AccessDeniedError struct {
}

func NewAccessDeniedError() *AccessDeniedError {
	return &AccessDeniedError{}
}

func (err *AccessDeniedError) Error() string {
	return T("Server error, status code: 403: Access is denied.  You do not have privileges to execute this command.")
}
