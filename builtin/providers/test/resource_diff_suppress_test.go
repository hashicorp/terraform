package test

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestResourceDiffSuppress_create(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_diff_suppress" "foo" {
	val_to_upper = "foo"
}
				`),
			},
		},
	})
}
func TestResourceDiffSuppress_update(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_diff_suppress" "foo" {
	val_to_upper = "foo"
}
				`),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_diff_suppress" "foo" {
	val_to_upper = "bar"
	optional = "more"
}
				`),
			},
		},
	})
}
