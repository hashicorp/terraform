package remotestate

import (
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state/remote"
)

func TestClient(t *testing.T, raw backend.Backend) {
	b, ok := raw.(*Backend)
	if !ok {
		t.Fatalf("not Backend: %T", raw)
	}

	remote.TestClient(t, b.client)
}
