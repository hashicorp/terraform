package consul

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/hashicorp/consul/testutil"
	"github.com/hashicorp/terraform/internal/backend"
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

		if !flag.Parsed() {
			flag.Parse()
		}

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

func TestBackend_encryption(t *testing.T) {
	path, err := exec.LookPath("vault")
	if err != nil {
		t.Skip("Install vault to run this test")
	}

	command := func(args []string) *exec.Cmd {
		return &exec.Cmd{
			Path: path,
			Args: args,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}
	}

	cmd := command([]string{"vault", "server", "-dev", "-dev-root-token-id=root-token"})
	if err = cmd.Start(); err != nil {
		t.Fatalf("failed to start vault server: %s", err)
	}
	defer cmd.Process.Kill()

	time.Sleep(1*time.Second)

	cmd = command([]string{"vault", "secrets", "enable", "-address=http://127.0.0.1:8200", "transit"})
	if err = cmd.Run(); err != nil {
		t.Fatalf("failed to mount transit secret engine: %s", err)
	}

	cmd = command([]string{"vault", "write", "-address=http://127.0.0.1:8200", "-f", "transit/keys/terraform"})
	if err = cmd.Run(); err != nil {
		t.Fatalf("failed to create transit key: %s", err)
	}

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"address": srv.HTTPAddr,
		"path": fmt.Sprintf("tf-unit/%s", time.Now().String()),
		"vault": []interface{}{map[string]interface{}{
				"address":  "http://localhost:8200",
				"token": "root-token",
				"key_name": "terraform",
			},},
	}))

	backend.TestBackendStates(t, b)
}
