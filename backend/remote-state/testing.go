package remotestate

import (
	"testing"

	"github.com/r3labs/terraform/backend"
	"github.com/r3labs/terraform/state/remote"
)

func TestClient(t *testing.T, raw backend.Backend) {
	b, ok := raw.(*Backend)
	if !ok {
		t.Fatalf("not Backend: %T", raw)
	}

	remote.TestClient(t, b.client)
}
