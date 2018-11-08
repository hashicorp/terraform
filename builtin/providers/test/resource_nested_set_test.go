package test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceNestedSet_basic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	single {
		value = "bar"
	}
}
				`),
			},
		},
	})
}

// The set should not be generated because of it's computed value
func TestResourceNestedSet_noSet(t *testing.T) {
	checkFunc := func(s *terraform.State) error {
		root := s.ModuleByPath(addrs.RootModuleInstance)
		res := root.Resources["test_resource_nested_set.foo"]
		for k, v := range res.Primary.Attributes {
			if strings.HasPrefix(k, "single") && k != "single.#" {
				return fmt.Errorf("unexpected set value: %s:%s", k, v)
			}
		}
		return nil
	}
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
}
				`),
				Check: checkFunc,
			},
		},
	})
}

func TestResourceNestedSet_addRemove(t *testing.T) {
	var id string
	checkFunc := func(s *terraform.State) error {
		root := s.ModuleByPath(addrs.RootModuleInstance)
		res := root.Resources["test_resource_nested_set.foo"]
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
resource "test_resource_nested_set" "foo" {
}
				`),
				Check: checkFunc,
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	single {
		value = "bar"
	}
}
				`),
				Check: checkFunc,
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
}
				`),
				Check: checkFunc,
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	single {
		value = "bar"
	}
}
				`),
				Check: checkFunc,
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
	single {
		value = "bar"
		optional = "baz"
	}
}
				`),
				Check: checkFunc,
			},

			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_set" "foo" {
}
			   	`),
				Check: checkFunc,
			},
		},
	})
}
