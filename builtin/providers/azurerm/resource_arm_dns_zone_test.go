package azurerm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/jen20/riviera/dns"
)

func TestAccAzureRMDnsZone_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMDnsZone_basic, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMDnsZoneDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDnsZoneExists("azurerm_dns_zone.test"),
				),
			},
		},
	})
}

func testCheckAzureRMDnsZoneExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).rivieraClient

		readRequest := conn.NewRequestForURI(rs.Primary.ID)
		readRequest.Command = &dns.GetDNSZone{}

		readResponse, err := readRequest.Execute()
		if err != nil {
			return fmt.Errorf("Bad: GetDNSZone: %s", err)
		}
		if !readResponse.IsSuccessful() {
			return fmt.Errorf("Bad: GetDNSZone: %s", readResponse.Error)
		}

		return nil
	}
}

func testCheckAzureRMDnsZoneDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).rivieraClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_dns_zone" {
			continue
		}

		readRequest := conn.NewRequestForURI(rs.Primary.ID)
		readRequest.Command = &dns.GetDNSZone{}

		readResponse, err := readRequest.Execute()
		if err != nil {
			return fmt.Errorf("Bad: GetDNSZone: %s", err)
		}

		if readResponse.IsSuccessful() {
			return fmt.Errorf("Bad: DNS zone still exists: %s", readResponse.Error)
		}
	}

	return nil
}

var testAccAzureRMDnsZone_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctest_rg_%d"
    location = "West US"
}
resource "azurerm_dns_zone" "test" {
    name = "acctestzone%d.com"
    resource_group_name = "${azurerm_resource_group.test.name}"
}
`
