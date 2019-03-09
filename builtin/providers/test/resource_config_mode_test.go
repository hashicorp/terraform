package test

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestResourceConfigMode(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_config_mode" "foo" {
	resource_as_attr = [
		{
			foo = "resource_as_attr 0"
		},
		{
			foo = "resource_as_attr 1"
		},
	]
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr.#", "2"),
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr.0.foo", "resource_as_attr 0"),
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr.1.foo", "resource_as_attr 1"),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_config_mode" "foo" {
	resource_as_attr = [
		{
			foo = "resource_as_attr 0 updated"
		},
	]
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr.#", "1"),
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr.0.foo", "resource_as_attr 0 updated"),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_config_mode" "foo" {
	resource_as_attr = []
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr.#", "0"),
				),
			},
		},
	})
}
