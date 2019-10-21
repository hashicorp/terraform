package test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

// an empty config should be ok, because no deprecated/removed fields are set.
func TestResourceDeprecated_empty(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_deprecated" "foo" {
}
				`),
			},
		},
	})
}

// Deprecated fields should still work
func TestResourceDeprecated_deprecatedOK(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_deprecated" "foo" {
	map_deprecated = {
		"a" = "b",
	}
	set_block_deprecated {
		value = "1"
	}
	list_block_deprecated {
		value = "2"
	}
}
				`),
			},
		},
	})
}

// Declaring an empty block should trigger the error
func TestResourceDeprecated_removedBlocks(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_deprecated" "foo" {
	set_block_removed {
	}
	list_block_removed {
	}
}
				`),
				ExpectError: regexp.MustCompile("REMOVED"),
			},
		},
	})
}
