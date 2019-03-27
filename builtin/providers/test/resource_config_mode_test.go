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
	resource_as_attr_dynamic = [
		{
			foo = "resource_as_attr_dynamic 0"
		},
		{
		},
	]
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr.#", "2"),
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr.0.foo", "resource_as_attr 0"),
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr.1.foo", "resource_as_attr 1"),
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr_dynamic.#", "2"),
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr_dynamic.0.foo", "resource_as_attr_dynamic 0"),
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr_dynamic.1.foo", "default"),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_config_mode" "foo" {
	# Due to a preprocessing fixup we do in lang.EvalBlock, it's allowed
	# to specify resource_as_attr members using one or more nested blocks
	# instead of attribute syntax, if desired. This should be equivalent
	# to the previous config.
	#
	# This allowance is made for backward-compatibility with existing providers
	# before Terraform v0.12 that were expecting nested block types to also
	# support attribute syntax; it should not be used for any new use-cases.
	resource_as_attr {
		foo = "resource_as_attr 0"
	}
	resource_as_attr {
		foo = "resource_as_attr 1"
	}
	resource_as_attr_dynamic = [
		{
			foo = "resource_as_attr_dynamic 0"
		},
		{
		},
	]
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr.#", "2"),
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr.0.foo", "resource_as_attr 0"),
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr.1.foo", "resource_as_attr 1"),
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr_dynamic.#", "2"),
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr_dynamic.0.foo", "resource_as_attr_dynamic 0"),
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr_dynamic.1.foo", "default"),
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
	resource_as_attr_dynamic = [
		{
		},
	]
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr.#", "1"),
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr.0.foo", "resource_as_attr 0 updated"),
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr_dynamic.#", "1"),
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr_dynamic.0.foo", "default"),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_config_mode" "foo" {
	resource_as_attr = []
	resource_as_attr_dynamic = []
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr.#", "0"),
					resource.TestCheckResourceAttr("test_resource_config_mode.foo", "resource_as_attr_dynamic.#", "0"),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_config_mode" "foo" {
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("test_resource_config_mode.foo", "resource_as_attr.#"),
					resource.TestCheckNoResourceAttr("test_resource_config_mode.foo", "resource_as_attr_dynamic.#"),
				),
			},
		},
	})
}
