package remote

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestConsulClient_impl(t *testing.T) {
	var _ Client = new(ConsulClient)
}

func TestConsulClient(t *testing.T) {
	if _, err := http.Get("http://google.com"); err != nil {
		t.Skipf("skipping, internet seems to not be available: %s", err)
	}

	client, err := consulFactory(map[string]string{
		"address": "demo.consul.io:80",
		"path":    fmt.Sprintf("tf-unit/%s", time.Now().String()),
	})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	testClient(t, client)
}
