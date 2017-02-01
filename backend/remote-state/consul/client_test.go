package consul

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/backend/remote-state"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/state/remote"
)

func TestRemoteClient_impl(t *testing.T) {
	var _ remote.Client = new(RemoteClient)
}

func TestRemoteClient(t *testing.T) {
	acctest.RemoteTestPrecheck(t)

	// Get the backend
	b := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"address": "demo.consul.io:80",
		"path":    fmt.Sprintf("tf-unit/%s", time.Now().String()),
	})

	// Test
	remotestate.TestClient(t, b)
}
