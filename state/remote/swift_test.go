package remote

import (
	"net/http"
	"os"
	"testing"
)

func TestSwiftClient_impl(t *testing.T) {
	var _ Client = new(SwiftClient)
}

func TestSwiftClient(t *testing.T) {
	os_auth_url := os.Getenv("OS_AUTH_URL")
	if os_auth_url == "" {
		t.Skipf("skipping, OS_AUTH_URL and friends must be set")
	}

	if _, err := http.Get(os_auth_url); err != nil {
		t.Skipf("skipping, unable to reach %s: %s", os_auth_url, err)
	}

	client, err := swiftFactory(map[string]string{
		"path": "swift_test",
	})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	testClient(t, client)
}
