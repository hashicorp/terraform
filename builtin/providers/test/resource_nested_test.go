package test

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceNested_basic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested" "foo" {
	nested {
		string = "val"
	}
}
				`),
			},
		},
	})
}

func TestResourceNested_addRemove(t *testing.T) {
	var id string
	checkFunc := func(s *terraform.State) error {
		root := s.ModuleByPath(addrs.RootModuleInstance)
		res := root.Resources["test_resource_nested.foo"]
		if res.Primary.ID == id {
			return errors.New("expected new resource")
		}
		id = res.Primary.ID
		return nil
	}
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested" "foo" {
}
				`),
				Check: checkFunc,
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested" "foo" {
	nested {
		string = "val"
	}
}
				`),
				Check: checkFunc,
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested" "foo" {
	optional = true
	nested {
		string = "val"
	}
}
				`),
				Check: checkFunc,
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested" "foo" {
	nested {
		string = "val"
	}
}
				`),
				Check: checkFunc,
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested" "foo" {
	nested {
		string = "val"
		optional = true
	}
}
				`),
				Check: checkFunc,
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested" "foo" {
}
				`),
				Check: checkFunc,
			},
		},
	})
}

func TestResourceNested_dynamic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested" "foo" {
	dynamic "nested" {
		for_each = [["a"], []]
		content {
			string   = join(",", nested.value)
			optional = false
			dynamic "nested_again" {
				for_each = nested.value
				content {
					string = nested_again.value
				}
			}
		}
	}
}
				`),
				Check: func(s *terraform.State) error {
					rs, ok := s.RootModule().Resources["test_resource_nested.foo"]
					if !ok {
						return errors.New("missing resource in state")
					}

					got := rs.Primary.Attributes
					want := map[string]string{
						"nested.#":                       "2",
						"nested.0.string":                "a",
						"nested.0.optional":              "false",
						"nested.0.nested_again.#":        "1",
						"nested.0.nested_again.0.string": "a",
						"nested.1.string":                "",
						"nested.1.optional":              "false",
						"nested.1.nested_again.#":        "0",
					}
					delete(got, "id") // it's random, so not useful for testing

					if !cmp.Equal(got, want) {
						return errors.New("wrong result\n" + cmp.Diff(want, got))
					}

					return nil
				},
			},
		},
	})
}
