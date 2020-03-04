package auth

import (
	"net/http"

	"github.com/zclconf/go-cty/cty"
)

// HostCredentialsToken is a HostCredentials implementation that represents a
// single "bearer token", to be sent to the server via an Authorization header
// with the auth type set to "Bearer".
//
// To save a token as the credentials for a host, convert the token string to
// this type and use the result as a HostCredentialsWritable implementation.
type HostCredentialsToken string

// Interface implementation assertions. Compilation will fail here if
// HostCredentialsToken does not fully implement these interfaces.
var _ HostCredentials = HostCredentialsToken("")
var _ HostCredentialsWritable = HostCredentialsToken("")

// PrepareRequest alters the given HTTP request by setting its Authorization
// header to the string "Bearer " followed by the encapsulated authentication
// token.
func (tc HostCredentialsToken) PrepareRequest(req *http.Request) {
	if req.Header == nil {
		req.Header = http.Header{}
	}
	req.Header.Set("Authorization", "Bearer "+string(tc))
}

// Token returns the authentication token.
func (tc HostCredentialsToken) Token() string {
	return string(tc)
}

// ToStore returns a credentials object with a single attribute "token" whose
// value is the token string.
func (tc HostCredentialsToken) ToStore() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"token": cty.StringVal(string(tc)),
	})
}
