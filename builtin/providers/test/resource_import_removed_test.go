package test

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestResourceImportRemoved(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_import_removed" "foo" {
}
				`),
			},
			{
				ImportState:  true,
				ResourceName: "test_resource_import_removed.foo",

				// This is attempting to guard against regressions of:
				// https://github.com/hashicorp/terraform/issues/20985
				//
				// Removed attributes are generally not populated during Create,
				// Update, Read, or Import by provider code but due to our
				// legacy diff format being lossy they end up getting populated
				// with zero values during shimming in all cases except Import,
				// which doesn't go through a diff.
				//
				// This is testing that the shimming inconsistency won't cause
				// ImportStateVerify failures for these, since we now ignore
				// attributes marked as Removed when comparing.
				ImportStateVerify: true,
			},
		},
	})
}
