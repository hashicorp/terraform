package circonus

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceCirconusAccount(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceCirconusAccountConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceCirconusAccountCheck("data.circonus_account.by_current", "/account/2438"),
				),
			},
		},
	})
}

func testAccDataSourceCirconusAccountCheck(name, cid string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		attr := rs.Primary.Attributes

		if attr["cid"] != cid {
			return fmt.Errorf("bad id %s", attr["cid"])
		}

		return nil
	}
}

const testAccDataSourceCirconusAccountConfig = `
variable circonus_api_token {}

provider "circonus" {
  key = "${var.circonus_api_token}"
}

data "circonus_account" "by_current" {
  current = true
}
`
