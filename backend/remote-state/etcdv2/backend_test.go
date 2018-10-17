package etcdv2

import (
	"testing"

	"github.com/hashicorp/terraform/backend"
)

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}
