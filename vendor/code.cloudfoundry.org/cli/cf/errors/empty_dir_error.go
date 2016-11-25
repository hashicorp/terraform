package errors

import (
	. "code.cloudfoundry.org/cli/cf/i18n"
)

type EmptyDirError struct {
	dir string
}

func NewEmptyDirError(dir string) error {
	return &EmptyDirError{dir: dir}
}

func (err *EmptyDirError) Error() string {
	return err.dir + T(" is empty")
}
