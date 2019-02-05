package test

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestResourceListSet_basic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list_set" "foo" {
  list {
    set {
      elem = "A"
    }
    set {
      elem = "B"
    }
  }
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test_resource_list_set.foo", "list.0.set.1255198513.elem", "B"),
					resource.TestCheckResourceAttr("test_resource_list_set.foo", "list.0.set.3554254475.elem", "A"),
					resource.TestCheckResourceAttr("test_resource_list_set.foo", "list.0.set.#", "2"),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list_set" "foo" {
  list {
    set {
      elem = "B"
    }
    set {
      elem = "C"
    }
  }
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test_resource_list_set.foo", "list.0.set.1255198513.elem", "B"),
					resource.TestCheckResourceAttr("test_resource_list_set.foo", "list.0.set.1037565863.elem", "C"),
					resource.TestCheckResourceAttr("test_resource_list_set.foo", "list.0.set.#", "2"),
				),
			},
		},
	})
}
