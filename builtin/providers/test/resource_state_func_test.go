package test

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestResourceStateFunc_basic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_state_func" "foo" {
}
				`),
				Check: resource.TestCheckNoResourceAttr("test_resource_state_func.foo", "state_func"),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_state_func" "foo" {
	state_func = "data"
	state_func_value = "data"
}
				`),
				Check: resource.TestCheckResourceAttr("test_resource_state_func.foo", "state_func", stateFuncHash("data")),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_state_func" "foo" {
}
				`),
				Check: resource.TestCheckNoResourceAttr("test_resource_state_func.foo", "state_func"),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_state_func" "foo" {
	optional = "added"
	state_func = "data"
	state_func_value = "data"
}
				`),
				Check: resource.TestCheckResourceAttr("test_resource_state_func.foo", "state_func", stateFuncHash("data")),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_state_func" "foo" {
	optional = "added"
	state_func = "changed"
	state_func_value = "changed"
}
				`),
				Check: resource.TestCheckResourceAttr("test_resource_state_func.foo", "state_func", stateFuncHash("changed")),
			},
		},
	})
}

func TestResourceStateFunc_getOkSetElem(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_state_func" "foo" {
}

resource "test_resource_state_func" "bar" {
	set_block {
		required = "foo"
		optional = test_resource_state_func.foo.id
	}
	set_block {
		required = test_resource_state_func.foo.id
	}
}
				`),
			},
		},
	})
}
