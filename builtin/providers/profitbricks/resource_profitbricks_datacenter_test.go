package profitbricks

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/profitbricks/profitbricks-sdk-go"
)

func TestAccProfitBricksDataCenter_Basic(t *testing.T) {
	var datacenter profitbricks.Datacenter
	lanName := "datacenter-test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDProfitBricksDatacenterDestroyCheck,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckProfitBricksDatacenterConfig_basic, lanName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProfitBricksDatacenterExists("profitbricks_datacenter.foobar", &datacenter),
					testAccCheckProfitBricksDatacenterAttributes("profitbricks_datacenter.foobar", lanName),
					resource.TestCheckResourceAttr("profitbricks_datacenter.foobar", "name", lanName),
				),
			},
			resource.TestStep{
				Config: testAccCheckProfitBricksDatacenterConfig_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProfitBricksDatacenterExists("profitbricks_datacenter.foobar", &datacenter),
					testAccCheckProfitBricksDatacenterAttributes("profitbricks_datacenter.foobar", "updated"),
					resource.TestCheckResourceAttr("profitbricks_datacenter.foobar", "name", "updated"),
				),
			},
		},
	})
}

func testAccCheckDProfitBricksDatacenterDestroyCheck(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "profitbricks_datacenter" {
			continue
		}

		resp := profitbricks.GetDatacenter(rs.Primary.ID)

		if resp.StatusCode < 299 {
			return fmt.Errorf("DataCenter still exists %s %s", rs.Primary.ID, resp.Response)
		}
	}

	return nil
}

func testAccCheckProfitBricksDatacenterAttributes(n string, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.Attributes["name"] != name {
			return fmt.Errorf("Bad name: expected %s : found %s ", name, rs.Primary.Attributes["name"])
		}

		return nil
	}
}

func testAccCheckProfitBricksDatacenterExists(n string, datacenter *profitbricks.Datacenter) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		foundDC := profitbricks.GetDatacenter(rs.Primary.ID)

		if foundDC.StatusCode != 200 {
			return fmt.Errorf("Error occured while fetching DC: %s", rs.Primary.ID)
		}
		if foundDC.Id != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}
		datacenter = &foundDC

		return nil
	}
}

const testAccCheckProfitBricksDatacenterConfig_basic = `
resource "profitbricks_datacenter" "foobar" {
	name       = "%s"
	location = "us/las"
}`

const testAccCheckProfitBricksDatacenterConfig_update = `
resource "profitbricks_datacenter" "foobar" {
	name       =  "updated"
	location = "us/las"
}`
