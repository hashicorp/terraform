package command

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestCountHook_impl(t *testing.T) {
	var _ terraform.Hook = new(CountHook)
}
