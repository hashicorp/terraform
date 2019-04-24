package test

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestResourceComputedSet_update(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_computed_set" "foo" {
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_computed_set.foo", "string_set.#", "3",
					),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_computed_set" "foo" {
	set_count = 5
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_computed_set.foo", "string_set.#", "5",
					),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_computed_set" "foo" {
	set_count = 2
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_computed_set.foo", "string_set.#", "2",
					),
				),
			},
		},
	})
}

func TestResourceComputedSet_ruleTest(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_computed_set" "foo" {
	rule {
		ip_protocol = "udp"
		cidr = "0.0.0.0/0"
	}
}
				`),
			},
		},
	})
}
