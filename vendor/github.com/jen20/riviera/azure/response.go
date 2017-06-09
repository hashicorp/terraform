package azure

import "net/http"

// Response is the type returned by API operations. It provides low
// level access to the HTTP request from the operation if that is required,
// and a parsed version of the response as an interface{} which can be
// type asserted to the correct response type for the request.
type Response struct {
	// HTTP provides direct access to the http.Response, though use should
	// not be necessary as a matter of course
	HTTP *http.Response

	// Parsed provides access the response structure of the command
	Parsed interface{}

	// Error provides access to the error body if the command was unsuccessful
	Error *Error
}

// IsSuccessful returns true if the status code for the underlying
// HTTP request was a "successful" status code.
func (response *Response) IsSuccessful() bool {
	return isSuccessCode(response.HTTP.StatusCode)
}
