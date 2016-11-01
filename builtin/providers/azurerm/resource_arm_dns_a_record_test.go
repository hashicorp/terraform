package azurerm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/jen20/riviera/dns"
)

func TestAccAzureRMDnsARecord_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMDnsARecord_basic, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMDnsARecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDnsARecordExists("azurerm_dns_a_record.test"),
				),
			},
		},
	})
}

func TestAccAzureRMDnsARecord_updateRecords(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMDnsARecord_basic, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMDnsARecord_updateRecords, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMDnsARecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDnsARecordExists("azurerm_dns_a_record.test"),
					resource.TestCheckResourceAttr(
						"azurerm_dns_a_record.test", "records.#", "2"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDnsARecordExists("azurerm_dns_a_record.test"),
					resource.TestCheckResourceAttr(
						"azurerm_dns_a_record.test", "records.#", "3"),
				),
			},
		},
	})
}

func TestAccAzureRMDnsARecord_withTags(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMDnsARecord_withTags, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMDnsARecord_withTagsUpdate, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMDnsARecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDnsARecordExists("azurerm_dns_a_record.test"),
					resource.TestCheckResourceAttr(
						"azurerm_dns_a_record.test", "tags.%", "2"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDnsARecordExists("azurerm_dns_a_record.test"),
					resource.TestCheckResourceAttr(
						"azurerm_dns_a_record.test", "tags.%", "1"),
				),
			},
		},
	})
}

func testCheckAzureRMDnsARecordExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).rivieraClient

		readRequest := conn.NewRequestForURI(rs.Primary.ID)
		readRequest.Command = &dns.GetARecordSet{}

		readResponse, err := readRequest.Execute()
		if err != nil {
			return fmt.Errorf("Bad: GetARecordSet: %s", err)
		}
		if !readResponse.IsSuccessful() {
			return fmt.Errorf("Bad: GetARecordSet: %s", readResponse.Error)
		}

		return nil
	}
}

func testCheckAzureRMDnsARecordDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).rivieraClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_dns_a_record" {
			continue
		}

		readRequest := conn.NewRequestForURI(rs.Primary.ID)
		readRequest.Command = &dns.GetARecordSet{}

		readResponse, err := readRequest.Execute()
		if err != nil {
			return fmt.Errorf("Bad: GetARecordSet: %s", err)
		}

		if readResponse.IsSuccessful() {
			return fmt.Errorf("Bad: DNS A Record still exists: %s", readResponse.Error)
		}
	}

	return nil
}

var testAccAzureRMDnsARecord_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG_%d"
    location = "West US"
}
resource "azurerm_dns_zone" "test" {
    name = "acctestzone%d.com"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_dns_a_record" "test" {
    name = "myarecord%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    zone_name = "${azurerm_dns_zone.test.name}"
    ttl = "300"
    records = ["1.2.3.4", "1.2.4.5"]
}
`

var testAccAzureRMDnsARecord_updateRecords = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG_%d"
    location = "West US"
}
resource "azurerm_dns_zone" "test" {
    name = "acctestzone%d.com"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_dns_a_record" "test" {
    name = "myarecord%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    zone_name = "${azurerm_dns_zone.test.name}"
    ttl = "300"
    records = ["1.2.3.4", "1.2.4.5", "1.2.3.7"]
}
`

var testAccAzureRMDnsARecord_withTags = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG_%d"
    location = "West US"
}
resource "azurerm_dns_zone" "test" {
    name = "acctestzone%d.com"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_dns_a_record" "test" {
    name = "myarecord%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    zone_name = "${azurerm_dns_zone.test.name}"
    ttl = "300"
    records = ["1.2.3.4", "1.2.4.5"]

    tags {
	environment = "Production"
	cost_center = "MSFT"
    }
}
`

var testAccAzureRMDnsARecord_withTagsUpdate = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG_%d"
    location = "West US"
}
resource "azurerm_dns_zone" "test" {
    name = "acctestzone%d.com"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_dns_a_record" "test" {
    name = "myarecord%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    zone_name = "${azurerm_dns_zone.test.name}"
    ttl = "300"
    records = ["1.2.3.4", "1.2.4.5"]

    tags {
	environment = "staging"
    }
}
`
