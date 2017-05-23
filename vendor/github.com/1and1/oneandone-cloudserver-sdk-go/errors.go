package oneandone

import (
	"fmt"
)

type errorResponse struct {
	Type    string      `json:"type"`
	Message string      `json:"message"`
}

type apiError struct {
	httpStatusCode int
	message        string
}

func (e apiError) Error() string {
	return fmt.Sprintf("%d - %s", e.httpStatusCode, e.message)
}

func (e *apiError) HttpStatusCode() int {
	return e.httpStatusCode
}

func (e *apiError) Message() string {
	return e.message
}
