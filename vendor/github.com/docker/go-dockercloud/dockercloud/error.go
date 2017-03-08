package dockercloud

import (
	"fmt"
)

type HttpError struct {
	Status     string
	StatusCode int
}

func (e HttpError) Error() string {
	return fmt.Sprintf("Failed API call: %s", e.Status)
}
