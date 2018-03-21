package etcdv2

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/zclconf/go-cty/cty"
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
	config := map[string]cty.Value{
		"endpoints": cty.StringVal(endpoint),
		"path":      cty.StringVal(fmt.Sprintf("tf-unit/%s", time.Now().String())),
	}

	if username := os.Getenv("ETCD_USERNAME"); username != "" {
		config["username"] = cty.StringVal(username)
	}
	if password := os.Getenv("ETCD_PASSWORD"); password != "" {
		config["password"] = cty.StringVal(password)
	}

	b := backend.TestBackendConfig(t, New(), configs.SynthBody("synth", config))
	state, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("Error for valid config: %s", err)
	}

	remote.TestClient(t, state.(*remote.State).Client)
}
