package test

import (
	"errors"
	"strings"
	"testing"

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
