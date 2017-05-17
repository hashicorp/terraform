package profitbricks

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/profitbricks/profitbricks-sdk-go"
)

func TestAccProfitBricksIPBlock_Basic(t *testing.T) {
	var ipblock profitbricks.IpBlock
	location := "us/las"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDProfitBricksIPBlockDestroyCheck,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckProfitbricksIPBlockConfig_basic, location),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProfitBricksIPBlockExists("profitbricks_ipblock.webserver_ip", &ipblock),
					testAccCheckProfitBricksIPBlockAttributes("profitbricks_ipblock.webserver_ip", location),
					resource.TestCheckResourceAttr("profitbricks_ipblock.webserver_ip", "location", location),
				),
			},
		},
	})
}

func testAccCheckDProfitBricksIPBlockDestroyCheck(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "profitbricks_ipblock" {
			continue
		}

		resp := profitbricks.GetIpBlock(rs.Primary.ID)

		if resp.StatusCode < 299 {
			return fmt.Errorf("IPBlock still exists %s %s", rs.Primary.ID, resp.Response)
		}
	}

	return nil
}

func testAccCheckProfitBricksIPBlockAttributes(n string, location string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("testAccCheckProfitBricksLanAttributes: Not found: %s", n)
		}
		if rs.Primary.Attributes["location"] != location {
			return fmt.Errorf("Bad name: %s", rs.Primary.Attributes["location"])
		}

		return nil
	}
}

func testAccCheckProfitBricksIPBlockExists(n string, ipblock *profitbricks.IpBlock) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("testAccCheckProfitBricksIPBlockExists: Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		foundIP := profitbricks.GetIpBlock(rs.Primary.ID)

		if foundIP.StatusCode != 200 {
			return fmt.Errorf("Error occured while fetching IP Block: %s", rs.Primary.ID)
		}
		if foundIP.Id != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		ipblock = &foundIP

		return nil
	}
}

const testAccCheckProfitbricksIPBlockConfig_basic = `
resource "profitbricks_ipblock" "webserver_ip" {
  location = "%s"
  size = 1
}`
