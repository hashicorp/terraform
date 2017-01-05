package errors

import (
	"fmt"

	. "code.cloudfoundry.org/cli/cf/i18n"
)

type AsyncTimeoutError struct {
	url string
}

func NewAsyncTimeoutError(url string) error {
	return &AsyncTimeoutError{url: url}
}

func (err *AsyncTimeoutError) Error() string {
	return fmt.Sprintf(T("Error: timed out waiting for async job '{{.ErrURL}}' to finish",
		map[string]interface{}{"ErrURL": err.url}))
}
