package errors

import original "errors"

func New(message string) error {
	return original.New(message)
}
