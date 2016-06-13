package spotinst

import (
	"fmt"
	"net/http"
	"strings"
)

const (
	apiURL    = "https://api.spotinst.io"
	oauthURL  = "https://oauth.spotinst.io"
	mediaType = "application/json"
	userAgent = SDKName + "/" + SDKVersion
)

type Credentials struct {
	Email        string `json:"username"`
	Password     string `json:"password"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Token        string `json:"token"`
}

type Response struct {
	Response struct {
		Errors []Error       `json:"errors"`
		Items  []interface{} `json:"items"`
	} `json:"response"`
}

type Error struct {
	// Error code.
	Code string `json:"code"`

	// Human-readable message.
	Message string `json:"message"`

	// The field in error.
	Field string `json:"field"`
}

// An ErrorResponse reports the errors caused by an API request.
type ErrorResponse struct {
	// HTTP response that caused this error.
	Response *http.Response

	// A list of errors.
	Errors []Error
}

// Error implements the error interface.
func (e *ErrorResponse) Error() string {
	if len(e.Errors) > 0 {
		errs := make([]string, len(e.Errors))
		for i, err := range e.Errors {
			serr := fmt.Sprintf("Method: %s, URL: %s, StatusCode: %d, ErrorCode: %s, Field: %s, Message: %s",
				e.Response.Request.Method,
				e.Response.Request.URL,
				e.Response.StatusCode,
				err.Code,
				err.Field,
				err.Message,
			)
			errs[i] = serr
		}
		return strings.Join(errs, "\n")
	} else {
		return fmt.Sprintf("Method: %s, URL: %s, StatusCode: %d",
			e.Response.Request.Method,
			e.Response.Request.URL,
			e.Response.StatusCode,
		)
	}
}