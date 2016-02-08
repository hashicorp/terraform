package packet

import (
	"net/http"
	"strings"

	"github.com/packethost/packngo"
)

func friendlyError(err error) error {
	if e, ok := err.(*packngo.ErrorResponse); ok {
		return &ErrorResponse{
			StatusCode: e.Response.StatusCode,
			Errors:     Errors(e.Errors),
		}
	}
	return err
}

func isForbidden(err error) bool {
	if r, ok := err.(*ErrorResponse); ok {
		return r.StatusCode == http.StatusForbidden
	}
	return false
}

func isNotFound(err error) bool {
	if r, ok := err.(*ErrorResponse); ok {
		return r.StatusCode == http.StatusNotFound
	}
	return false
}

type Errors []string

func (e Errors) Error() string {
	return strings.Join(e, "; ")
}

type ErrorResponse struct {
	StatusCode int
	Errors
}
