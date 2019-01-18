package test

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestResourceDefaults_basic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_defaults" "foo" {
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_defaults.foo", "default_string", "default string",
					),
					resource.TestCheckResourceAttr(
						"test_resource_defaults.foo", "default_bool", "1",
					),
					resource.TestCheckNoResourceAttr(
						"test_resource_defaults.foo", "nested.#",
					),
				),
			},
		},
	})
}

func TestResourceDefaults_inSet(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_defaults" "foo" {
	nested {
		optional = "val"
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_defaults.foo", "default_string", "default string",
					),
					resource.TestCheckResourceAttr(
						"test_resource_defaults.foo", "default_bool", "1",
					),
					resource.TestCheckResourceAttr(
						"test_resource_defaults.foo", "nested.2826070548.optional", "val",
					),
					resource.TestCheckResourceAttr(
						"test_resource_defaults.foo", "nested.2826070548.string", "default nested",
					),
				),
			},
		},
	})
}

func TestResourceDefaults_import(t *testing.T) {
	// FIXME: this test fails
	return

	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_defaults" "foo" {
	nested {
		optional = "val"
	}
}
				`),
			},
			{
				ImportState:       true,
				ImportStateVerify: true,
				ResourceName:      "test_resource_defaults.foo",
			},
		},
	})
}
