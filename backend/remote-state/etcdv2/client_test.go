package etcdv2

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state/remote"
)

func TestEtcdClient_impl(t *testing.T) {
	var _ remote.Client = new(EtcdClient)
}

func TestEtcdClient(t *testing.T) {
	endpoint := os.Getenv("ETCD_ENDPOINT")
	if endpoint == "" {
		t.Skipf("skipping; ETCD_ENDPOINT must be set")
	}

	// Get the backend
	config := map[string]interface{}{
		"endpoints": endpoint,
		"path":      fmt.Sprintf("tf-unit/%s", time.Now().String()),
	}

	if username := os.Getenv("ETCD_USERNAME"); username != "" {
		config["username"] = username
	}
	if password := os.Getenv("ETCD_PASSWORD"); password != "" {
		config["password"] = password
	}

	b := backend.TestBackendConfig(t, New(), config)
	state, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("Error for valid config: %s", err)
	}

	remote.TestClient(t, state.(*remote.State).Client)
}
