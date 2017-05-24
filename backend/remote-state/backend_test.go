package remotestate

import (
	"testing"

	"github.com/r3labs/terraform/backend"
)

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}
