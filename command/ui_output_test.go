package command

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestUIOutput_impl(t *testing.T) {
	var _ terraform.UIOutput = new(UIOutput)
}
