package circonus

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceCirconusCollector(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceCirconusCollectorConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceCirconusCollectorCheck("data.circonus_collector.by_cid", "/broker/1"),
				),
			},
		},
	})
}

func testAccDataSourceCirconusCollectorCheck(name, cid string) resource.TestCheckFunc {
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

const testAccDataSourceCirconusCollectorConfig = `
variable circonus_api_token {}

provider "circonus" {
  key = "${var.circonus_api_token}"
}

data "circonus_collector" "by_cid" {
  cid = "/broker/1"
}
`
