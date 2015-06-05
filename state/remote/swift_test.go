package remote

import (
	"net/http"
	"testing"
)

func TestSwiftClient_impl(t *testing.T) {
	var _ Client = new(SwiftClient)
}

func TestSwiftClient(t *testing.T) {
	if _, err := http.Get("http://google.com"); err != nil {
		t.Skipf("skipping, internet seems to not be available: %s", err)
	}

	client, err := swiftFactory(map[string]string{
		"path": "swift_test",
	})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	testClient(t, client)
}
