package test

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestResource_basic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required = "yep"
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
		},
	})
}

func testAccCheckResourceDestroy(s *terraform.State) error {
	return nil
}
