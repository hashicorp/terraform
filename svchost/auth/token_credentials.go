package auth

import (
	"net/http"
)

// HostCredentialsToken is a HostCredentials implementation that represents a
// single "bearer token", to be sent to the server via an Authorization header
// with the auth type set to "Bearer"
type HostCredentialsToken string

// PrepareRequest alters the given HTTP request by setting its Authorization
// header to the string "Bearer " followed by the encapsulated authentication
// token.
func (tc HostCredentialsToken) PrepareRequest(req *http.Request) {
	if req.Header == nil {
		req.Header = http.Header{}
	}
	req.Header.Set("Authorization", "Bearer "+string(tc))
}
