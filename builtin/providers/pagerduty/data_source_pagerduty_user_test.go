package pagerduty

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourcePagerDutyUser_Basic(t *testing.T) {
	username := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email := fmt.Sprintf("%s@foo.com", username)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePagerDutyUserConfig(username, email),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourcePagerDutyUser("pagerduty_user.test", "data.pagerduty_user.by_email"),
				),
			},
		},
	})
}

func testAccDataSourcePagerDutyUser(src, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		srcR := s.RootModule().Resources[src]
		srcA := srcR.Primary.Attributes

		r := s.RootModule().Resources[n]
		a := r.Primary.Attributes

		if a["id"] == "" {
			return fmt.Errorf("Expected to get a user ID from PagerDuty")
		}

		testAtts := []string{"id", "name", "email"}

		for _, att := range testAtts {
			if a[att] != srcA[att] {
				return fmt.Errorf("Expected the user %s to be: %s, but got: %s", att, srcA[att], a[att])
			}
		}

		return nil
	}
}

func testAccDataSourcePagerDutyUserConfig(username, email string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "test" {
  name = "%s"
  email = "%s"
}

data "pagerduty_user" "by_email" {
	email = "${pagerduty_user.test.email}"
}
`, username, email)
}
