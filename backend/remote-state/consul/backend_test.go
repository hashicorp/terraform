package consul

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform/backend"
)

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestBackend(t *testing.T) {
	addr := os.Getenv("CONSUL_HTTP_ADDR")
	if addr == "" {
		t.Log("consul tests require CONSUL_HTTP_ADDR")
		t.Skip()
	}

	// Get the backend
	b := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"address": addr,
		"path":    fmt.Sprintf("tf-unit/%s", time.Now().String()),
	})

	// Test
	backend.TestBackend(t, b)
}
