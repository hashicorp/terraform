package test

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestResourceComputedMap_basic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_computed_map" "foo" {
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_computed_map.foo", "nested_resources.#", "1",
					),
				),
			},
		},
	})
}
