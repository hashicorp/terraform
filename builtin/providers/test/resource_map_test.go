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

func TestResourceMap_basicWithVars(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: `
variable "a" {
  default = "a"
}

variable "b" {
  default = "b"
}

resource "test_resource_map" "foobar" {
	name = "test"
	map_of_three = {
		one   = var.a
		two   = var.b
		empty = ""
	}
}`,
				Check: resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func TestResourceMap_computedMap(t *testing.T) {
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
	map_values = {
		a = "1"
		b = "2"
	}
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_map.foobar", "computed_map.a", "1",
					),
					resource.TestCheckResourceAttr(
						"test_resource_map.foobar", "computed_map.b", "2",
					),
				),
			},
			{
				Config: `
resource "test_resource_map" "foobar" {
	name = "test"
	map_of_three = {
		one   = "one"
		two   = "two"
		empty = ""
	}
	map_values = {
		a = "3"
		b = "4"
	}
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_map.foobar", "computed_map.a", "3",
					),
					resource.TestCheckResourceAttr(
						"test_resource_map.foobar", "computed_map.b", "4",
					),
				),
			},
			{
				Config: `
resource "test_resource_map" "foobar" {
	name = "test"
	map_of_three = {
		one   = "one"
		two   = "two"
		empty = ""
	}
	map_values = {
		a = "3"
	}
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_map.foobar", "computed_map.a", "3",
					),
					resource.TestCheckNoResourceAttr(
						"test_resource_map.foobar", "computed_map.b",
					),
				),
			},
		},
	})
}
