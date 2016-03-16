package azurerm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/jen20/riviera/dns"
)

func TestAccAzureRMDnsCNameRecord_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMDnsCNameRecord_basic, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMDnsCNameRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDnsCNameRecordExists("azurerm_dns_cname_record.test"),
				),
			},
		},
	})
}

func TestAccAzureRMDnsCNameRecord_updateRecords(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMDnsCNameRecord_basic, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMDnsCNameRecord_updateRecords, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMDnsCNameRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDnsCNameRecordExists("azurerm_dns_cname_record.test"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDnsCNameRecordExists("azurerm_dns_cname_record.test"),
				),
			},
		},
	})
}

func TestAccAzureRMDnsCNameRecord_withTags(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMDnsCNameRecord_withTags, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMDnsCNameRecord_withTagsUpdate, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMDnsCNameRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDnsCNameRecordExists("azurerm_dns_cname_record.test"),
					resource.TestCheckResourceAttr(
						"azurerm_dns_cname_record.test", "tags.#", "2"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDnsCNameRecordExists("azurerm_dns_cname_record.test"),
					resource.TestCheckResourceAttr(
						"azurerm_dns_cname_record.test", "tags.#", "1"),
				),
			},
		},
	})
}

func testCheckAzureRMDnsCNameRecordExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).rivieraClient

		readRequest := conn.NewRequestForURI(rs.Primary.ID)
		readRequest.Command = &dns.GetCNAMERecordSet{}

		readResponse, err := readRequest.Execute()
		if err != nil {
			return fmt.Errorf("Bad: GetCNAMERecordSet: %s", err)
		}
		if !readResponse.IsSuccessful() {
			return fmt.Errorf("Bad: GetCNAMERecordSet: %s", readResponse.Error)
		}

		return nil
	}
}

func testCheckAzureRMDnsCNameRecordDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).rivieraClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_dns_cname_record" {
			continue
		}

		readRequest := conn.NewRequestForURI(rs.Primary.ID)
		readRequest.Command = &dns.GetCNAMERecordSet{}

		readResponse, err := readRequest.Execute()
		if err != nil {
			return fmt.Errorf("Bad: GetCNAMERecordSet: %s", err)
		}

		if readResponse.IsSuccessful() {
			return fmt.Errorf("Bad: DNS CNAME Record still exists: %s", readResponse.Error)
		}
	}

	return nil
}

var testAccAzureRMDnsCNameRecord_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctest_rg_%d"
    location = "West US"
}
resource "azurerm_dns_zone" "test" {
    name = "acctestzone%d.com"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_dns_cname_record" "test" {
    name = "myarecord%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    zone_name = "${azurerm_dns_zone.test.name}"
    ttl = "300"
    records = ["contoso.com"]
}
`

var testAccAzureRMDnsCNameRecord_updateRecords = `
resource "azurerm_resource_group" "test" {
    name = "acctest_rg_%d"
    location = "West US"
}
resource "azurerm_dns_zone" "test" {
    name = "acctestzone%d.com"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_dns_cname_record" "test" {
    name = "myarecord%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    zone_name = "${azurerm_dns_zone.test.name}"
    ttl = "300"
    records = ["contoso.co.uk"]
}
`

var testAccAzureRMDnsCNameRecord_withTags = `
resource "azurerm_resource_group" "test" {
    name = "acctest_rg_%d"
    location = "West US"
}
resource "azurerm_dns_zone" "test" {
    name = "acctestzone%d.com"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_dns_cname_record" "test" {
    name = "myarecord%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    zone_name = "${azurerm_dns_zone.test.name}"
    ttl = "300"
    records = ["contoso.com"]

    tags {
	environment = "Production"
	cost_center = "MSFT"
    }
}
`

var testAccAzureRMDnsCNameRecord_withTagsUpdate = `
resource "azurerm_resource_group" "test" {
    name = "acctest_rg_%d"
    location = "West US"
}
resource "azurerm_dns_zone" "test" {
    name = "acctestzone%d.com"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_dns_cname_record" "test" {
    name = "myarecord%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    zone_name = "${azurerm_dns_zone.test.name}"
    ttl = "300"
    records = ["contoso.com"]

    tags {
	environment = "staging"
    }
}
`
