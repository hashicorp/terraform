package test

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestResourceProviderMeta_basic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
terraform {
  provider_meta "test" {
    foo = "bar"
  }
}

resource "test_resource_provider_meta" "foo" {
}
				`),
			},
		},
	})
}
