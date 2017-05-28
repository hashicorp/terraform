package test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

// TestResourceWithCustomDiff test custom diff behaviour.
func TestResourceWithCustomDiff(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: resourceWithCustomDiffConfig(false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test_resource_with_custom_diff.foo", "computed", "1"),
					resource.TestCheckResourceAttr("test_resource_with_custom_diff.foo", "index", "1"),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: resourceWithCustomDiffConfig(false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test_resource_with_custom_diff.foo", "computed", "2"),
					resource.TestCheckResourceAttr("test_resource_with_custom_diff.foo", "index", "2"),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config:      resourceWithCustomDiffConfig(true),
				ExpectError: regexp.MustCompile("veto is true, diff vetoed"),
			},
		},
	})
}

func resourceWithCustomDiffConfig(veto bool) string {
	return fmt.Sprintf(`
resource "test_resource_with_custom_diff" "foo" {
	required = "yep"
	veto = %t
}
`, veto)
}
