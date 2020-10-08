package test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestResource_dynamicRequiredMinItems(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
resource "test_resource_required_min" "a" {
}
`,
				ExpectError: regexp.MustCompile(`"required_min_items" blocks are required`),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list" "a" {
	dependent_list {
		val = "a"
	}
}

resource "test_resource_required_min" "b" {
	dynamic "required_min_items" {
		for_each = test_resource_list.a.computed_list
		content {
		  val = required_min_items.value
		}
	}
}
				`),
				ExpectError: regexp.MustCompile(`required_min_items: attribute supports 2 item as a minimum`),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list" "c" {
	dependent_list {
		val = "a"
	}

	dependent_list {
		val = "b"
	}
}

resource "test_resource_required_min" "b" {
	dynamic "required_min_items" {
		for_each = test_resource_list.c.computed_list
		content {
		  val = required_min_items.value
		}
	}
}
				`),
			},
		},
	})
}
