package test

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceAsSingle(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_as_single" "foo" {
	list_resource_as_block {
		foo = "as block a"
	}
	list_resource_as_attr = {
		foo = "as attr a"
	}
	list_primitive = "primitive a"

	set_resource_as_block {
		foo = "as block a"
	}
	set_resource_as_attr = {
		foo = "as attr a"
	}
	set_primitive = "primitive a"
}
				`),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						t.Log("state after initial create:\n", s.String())
						return nil
					},
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "list_resource_as_block.#", "1"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "list_resource_as_block.0.foo", "as block a"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "list_resource_as_attr.#", "1"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "list_resource_as_attr.0.foo", "as attr a"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "list_primitive.#", "1"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "list_primitive.0", "primitive a"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "set_resource_as_block.#", "1"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "set_resource_as_block.1417230722.foo", "as block a"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "set_resource_as_attr.#", "1"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "set_resource_as_attr.2549052262.foo", "as attr a"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "set_primitive.#", "1"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "set_primitive.247272358", "primitive a"),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_as_single" "foo" {
	list_resource_as_block {
		foo = "as block b"
	}
	list_resource_as_attr = {
			foo = "as attr b"
	}
	list_primitive = "primitive b"

	set_resource_as_block {
		foo = "as block b"
	}
	set_resource_as_attr = {
		foo = "as attr b"
	}
	set_primitive = "primitive b"
}
				`),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						t.Log("state after update:\n", s.String())
						return nil
					},
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "list_resource_as_block.#", "1"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "list_resource_as_block.0.foo", "as block b"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "list_resource_as_attr.#", "1"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "list_resource_as_attr.0.foo", "as attr b"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "list_primitive.#", "1"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "list_primitive.0", "primitive b"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "set_resource_as_block.#", "1"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "set_resource_as_block.2136238657.foo", "as block b"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "set_resource_as_attr.#", "1"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "set_resource_as_attr.3166838949.foo", "as attr b"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "set_primitive.#", "1"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "set_primitive.630210661", "primitive b"),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_as_single" "foo" {
}
				`),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						t.Log("state after everything unset:\n", s.String())
						return nil
					},
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "list_resource_as_block.#", "0"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "list_resource_as_attr.#", "0"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "list_primitive.#", "0"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "set_resource_as_block.#", "0"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "set_resource_as_attr.#", "0"),
					resource.TestCheckResourceAttr("test_resource_as_single.foo", "set_primitive.#", "0"),
				),
			},
		},
	})
}
