package auth

import (
	"net/http"
	"testing"
)

func TestHostCredentialsToken(t *testing.T) {
	creds := HostCredentialsToken("foo-bar")
	req := &http.Request{}

	creds.PrepareRequest(req)

	authStr := req.Header.Get("authorization")
	if got, want := authStr, "Bearer foo-bar"; got != want {
		t.Errorf("wrong Authorization header value %q; want %q", got, want)
	}
}
