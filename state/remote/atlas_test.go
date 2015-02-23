package remote

import (
	"net/http"
	"os"
	"testing"
)

func TestAtlasClient_impl(t *testing.T) {
	var _ Client = new(AtlasClient)
}

func TestAtlasClient(t *testing.T) {
	if _, err := http.Get("http://google.com"); err != nil {
		t.Skipf("skipping, internet seems to not be available: %s", err)
	}

	token := os.Getenv("ATLAS_TOKEN")
	if token == "" {
		t.Skipf("skipping, ATLAS_TOKEN must be set")
	}

	client, err := atlasFactory(map[string]string{
		"access_token": token,
		"name":         "hashicorp/test-remote-state",
	})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	testClient(t, client)
}
