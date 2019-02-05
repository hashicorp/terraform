package test

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestResourceNestedId_unknownId(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_nested_id" "foo" {
}
resource "test_resource_nested_id" "bar" {
	list_block {
		id = test_resource_nested_id.foo.id
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test_resource_nested_id.bar", "list_block.0.id", "testId"),
				),
			},
		},
	})
}
