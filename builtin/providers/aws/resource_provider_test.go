package aws

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestResourceProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = new(ResourceProvider)
}
