package remote

import (
	"testing"
)

func TestHTTPClient_impl(t *testing.T) {
	var _ Client = new(HTTPClient)
}

func TestHTTPClient(t *testing.T) {
	// TODO
	//testClient(t, client)
}
