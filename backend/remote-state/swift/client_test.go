package swift

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state/remote"
)

func TestRemoteClient_impl(t *testing.T) {
	var _ remote.Client = new(RemoteClient)
}

func TestRemoteClient(t *testing.T) {
	testACC(t)

	container := fmt.Sprintf("terraform-state-swift-test-%x", time.Now().Unix())

	b := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"container": container,
	}).(*Backend)

	state, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	defer deleteSwiftContainer(t, b.client, container)

	remote.TestClient(t, state.(*remote.State).Client)
}
