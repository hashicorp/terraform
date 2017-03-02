package atlas

import (
	"testing"

	"github.com/hashicorp/terraform/backend"
)

func TestImpl(t *testing.T) {
	var _ backend.Backend = new(Backend)
	var _ backend.CLI = new(Backend)
}
