package statuscake

import (
	"fmt"
	"strings"
)

// APIError implements the error interface an it's used when the API response has errors.
type APIError interface {
	APIError() string
}

type httpError struct {
	status     string
	statusCode int
}

func (e *httpError) Error() string {
	return fmt.Sprintf("HTTP error: %d - %s", e.statusCode, e.status)
}

// ValidationError is a map where the key is the invalid field and the value is a message describing why the field is invalid.
type ValidationError map[string]string

func (e ValidationError) Error() string {
	var messages []string

	for k, v := range e {
		m := fmt.Sprintf("%s %s", k, v)
		messages = append(messages, m)
	}

	return strings.Join(messages, ", ")
}

type updateError struct {
	Issues interface{}
}

func (e *updateError) Error() string {
	var messages []string

	if issues, ok := e.Issues.(map[string]interface{}); ok {
		for k, v := range issues {
			m := fmt.Sprintf("%s %s", k, v)
			messages = append(messages, m)
		}
	} else if issues, ok := e.Issues.([]interface{}); ok {
		for _, v := range issues {
			m := fmt.Sprint(v)
			messages = append(messages, m)
		}
	}

	return strings.Join(messages, ", ")
}

// APIError returns the error specified in the API response
func (e *updateError) APIError() string {
	return e.Error()
}

type deleteError struct {
	Message string
}

func (e *deleteError) Error() string {
	return e.Message
}

// AuthenticationError implements the error interface and it's returned
// when API responses have authentication errors
type AuthenticationError struct {
	errNo   int
	message string
}

func (e *AuthenticationError) Error() string {
	return fmt.Sprintf("%d, %s", e.errNo, e.message)
}
