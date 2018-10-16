package test

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

// Tests GH-12183. This would previously cause a crash. More granular
// unit tests are scattered through helper/schema and terraform core for
// this.
func TestResourceGH12183_basic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_gh12183" "a" {
	config {
		name = "hello"
	}
}

resource "test_resource_gh12183" "b" {
	key = "${lookup(test_resource_gh12183.a.config[0], "name")}"
	config {
		name = "required"
	}
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
		},
	})
}
