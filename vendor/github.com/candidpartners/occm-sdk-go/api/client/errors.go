// Package implements helper client functionality
package client

// Error messages
const (
  ErrClientCreationFailed = "Client creation failed"
  ErrInvalidContext = "Invalid context"
	ErrInvalidCredentials = "Invalid user and/or password"
  ErrUnauthorized = "Not authorized"
  ErrForbidden = "Access forbidden"
  ErrInvalidHost = "Invalid host"
  ErrServerError = "Server error %s"
  ErrJSONConversion = "Error converting JSON"
  ErrCreatingHttpRequestForUri = "Error creating HTTP request for URI %s"
  ErrInvokingHttpRequestForUri = "Error invoking HTTP request for URI %s"
  ErrReadingResponseBody = "Error reading response body"
  ErrUnexpectedHttpResponse = "Unexpected HTTP response, status: %d, body: %q"
  ErrInvalidRequest = "Invalid request"
)
