package consul

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/hashicorp/terraform/internal/backend"
)

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func newConsulTestServer(t *testing.T) *testutil.TestServer {
	if os.Getenv("TF_ACC") == "" && os.Getenv("TF_CONSUL_TEST") == "" {
		t.Skipf("consul server tests require setting TF_ACC or TF_CONSUL_TEST")
	}

	srv, err := testutil.NewTestServerConfigT(t, func(c *testutil.TestServerConfig) {
		c.LogLevel = "warn"

		if !flag.Parsed() {
			flag.Parse()
		}

		if !testing.Verbose() {
			c.Stdout = ioutil.Discard
			c.Stderr = ioutil.Discard
		}
	})

	if err != nil {
		t.Fatalf("failed to create consul test server: %s", err)
	}

	srv.WaitForSerfCheck(t)
	srv.WaitForLeader(t)

	return srv
}

func TestBackend(t *testing.T) {
	srv := newConsulTestServer(t)

	path := fmt.Sprintf("tf-unit/%s", time.Now().String())

	// Get the backend. We need two to test locking.
	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"address": srv.HTTPAddr,
		"path":    path,
	}))

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"address": srv.HTTPAddr,
		"path":    path,
	}))

	// Test
	backend.TestBackendStates(t, b1)
	backend.TestBackendStateLocks(t, b1, b2)
}

func TestBackend_lockDisabled(t *testing.T) {
	srv := newConsulTestServer(t)

	path := fmt.Sprintf("tf-unit/%s", time.Now().String())

	// Get the backend. We need two to test locking.
	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"address": srv.HTTPAddr,
		"path":    path,
		"lock":    false,
	}))

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"address": srv.HTTPAddr,
		"path":    path + "different", // Diff so locking test would fail if it was locking
		"lock":    false,
	}))

	// Test
	backend.TestBackendStates(t, b1)
	backend.TestBackendStateLocks(t, b1, b2)
}

func TestBackend_gzip(t *testing.T) {
	srv := newConsulTestServer(t)

	// Get the backend
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"address": srv.HTTPAddr,
		"path":    fmt.Sprintf("tf-unit/%s", time.Now().String()),
		"gzip":    true,
	}))

	// Test
	backend.TestBackendStates(t, b)
}
