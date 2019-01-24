package test

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestResourceMap_basic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: `
resource "test_resource_map" "foobar" {
	name = "test"
	map_of_three = {
		one   = "one"
		two   = "two"
		empty = ""
	}
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_map.foobar", "map_of_three.empty", "",
					),
				),
			},
		},
	})
}
