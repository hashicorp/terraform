package opsgenie

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceOpsGenieUser_Basic(t *testing.T) {
	ri := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceOpsGenieUserConfig(ri),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceOpsGenieUser("opsgenie_user.test", "data.opsgenie_user.by_username"),
				),
			},
		},
	})
}

func testAccDataSourceOpsGenieUser(src, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		srcR := s.RootModule().Resources[src]
		srcA := srcR.Primary.Attributes

		r := s.RootModule().Resources[n]
		a := r.Primary.Attributes

		if a["id"] == "" {
			return fmt.Errorf("Expected to get a user ID from OpsGenie")
		}

		testAtts := []string{"username", "full_name", "role"}

		for _, att := range testAtts {
			if a[att] != srcA[att] {
				return fmt.Errorf("Expected the user %s to be: %s, but got: %s", att, srcA[att], a[att])
			}
		}

		return nil
	}
}

func testAccDataSourceOpsGenieUserConfig(ri int) string {
	return fmt.Sprintf(`
resource "opsgenie_user" "test" {
  username  = "acctest-%d@example.tld"
  full_name = "Acceptance Test User"
  role      = "User"
}

data "opsgenie_user" "by_username" {
  username = "${opsgenie_user.test.username}"
}
`, ri)
}
