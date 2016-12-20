package pagerduty

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourcePagerDutyUser_Basic(t *testing.T) {
	rName := acctest.RandString(5)
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourcePagerDutyUserConfig(rName),
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

func testAccDataSourcePagerDutyUserConfig(rName string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "test" {
  name = "TF User %[1]s"
  email = "tf.%[1]s@example.com"
}

data "pagerduty_user" "by_email" {
	email = "${pagerduty_user.test.email}"
}
`, rName)
}
