package consul

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/consul/testutil"
	"github.com/hashicorp/terraform/backend"
)

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

var srv *testutil.TestServer

func TestMain(m *testing.M) {
	if os.Getenv("TF_ACC") == "" && os.Getenv("TF_CONSUL_TEST") == "" {
		fmt.Println("consul server tests require setting TF_ACC or TF_CONSUL_TEST")
		return
	}

	var err error
	srv, err = newConsulTestServer()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	rc := m.Run()
	srv.Stop()
	os.Exit(rc)
}

func newConsulTestServer() (*testutil.TestServer, error) {
	srv, err := testutil.NewTestServerConfig(func(c *testutil.TestServerConfig) {
		c.LogLevel = "warn"

		if !testing.Verbose() {
			c.Stdout = ioutil.Discard
			c.Stderr = ioutil.Discard
		}
	})

	return srv, err
}

func TestBackend(t *testing.T) {
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
	// Get the backend
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"address": srv.HTTPAddr,
		"path":    fmt.Sprintf("tf-unit/%s", time.Now().String()),
		"gzip":    true,
	}))

	// Test
	backend.TestBackendStates(t, b)
}
