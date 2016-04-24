package azurerm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/jen20/riviera/dns"
)

func TestAccAzureRMDnsNsRecord_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMDnsNsRecord_basic, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMDnsNsRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDnsNsRecordExists("azurerm_dns_ns_record.test"),
				),
			},
		},
	})
}

func TestAccAzureRMDnsNsRecord_updateRecords(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMDnsNsRecord_basic, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMDnsNsRecord_updateRecords, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMDnsNsRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDnsNsRecordExists("azurerm_dns_ns_record.test"),
					resource.TestCheckResourceAttr(
						"azurerm_dns_ns_record.test", "record.#", "2"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDnsNsRecordExists("azurerm_dns_ns_record.test"),
					resource.TestCheckResourceAttr(
						"azurerm_dns_ns_record.test", "record.#", "3"),
				),
			},
		},
	})
}

func TestAccAzureRMDnsNsRecord_withTags(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMDnsNsRecord_withTags, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMDnsNsRecord_withTagsUpdate, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMDnsNsRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDnsNsRecordExists("azurerm_dns_ns_record.test"),
					resource.TestCheckResourceAttr(
						"azurerm_dns_ns_record.test", "tags.#", "2"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDnsNsRecordExists("azurerm_dns_ns_record.test"),
					resource.TestCheckResourceAttr(
						"azurerm_dns_ns_record.test", "tags.#", "1"),
				),
			},
		},
	})
}

func testCheckAzureRMDnsNsRecordExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).rivieraClient

		readRequest := conn.NewRequestForURI(rs.Primary.ID)
		readRequest.Command = &dns.GetNSRecordSet{}

		readResponse, err := readRequest.Execute()
		if err != nil {
			return fmt.Errorf("Bad: GetNSRecordSet: %s", err)
		}
		if !readResponse.IsSuccessful() {
			return fmt.Errorf("Bad: GetNSRecordSet: %s", readResponse.Error)
		}

		return nil
	}
}

func testCheckAzureRMDnsNsRecordDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).rivieraClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_dns_ns_record" {
			continue
		}

		readRequest := conn.NewRequestForURI(rs.Primary.ID)
		readRequest.Command = &dns.GetNSRecordSet{}

		readResponse, err := readRequest.Execute()
		if err != nil {
			return fmt.Errorf("Bad: GetNSRecordSet: %s", err)
		}

		if readResponse.IsSuccessful() {
			return fmt.Errorf("Bad: DNS NS Record still exists: %s", readResponse.Error)
		}
	}

	return nil
}

var testAccAzureRMDnsNsRecord_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctest_rg_%d"
    location = "West US"
}
resource "azurerm_dns_zone" "test" {
    name = "acctestzone%d.com"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_dns_ns_record" "test" {
    name = "myarecord%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    zone_name = "${azurerm_dns_zone.test.name}"
    ttl = "300"

    record {
    	nsdname = "ns1.contoso.com"
    }

    record {
    	nsdname = "ns2.contoso.com"
    }
}
`

var testAccAzureRMDnsNsRecord_updateRecords = `
resource "azurerm_resource_group" "test" {
    name = "acctest_rg_%d"
    location = "West US"
}
resource "azurerm_dns_zone" "test" {
    name = "acctestzone%d.com"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_dns_ns_record" "test" {
    name = "myarecord%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    zone_name = "${azurerm_dns_zone.test.name}"
    ttl = "300"

    record {
    	nsdname = "ns1.contoso.com"
    }

    record {
    	nsdname = "ns2.contoso.com"
    }

    record {
    	nsdname = "ns3.contoso.com"
    }
}
`

var testAccAzureRMDnsNsRecord_withTags = `
resource "azurerm_resource_group" "test" {
    name = "acctest_rg_%d"
    location = "West US"
}
resource "azurerm_dns_zone" "test" {
    name = "acctestzone%d.com"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_dns_ns_record" "test" {
    name = "myarecord%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    zone_name = "${azurerm_dns_zone.test.name}"
    ttl = "300"

    record {
    	nsdname = "ns1.contoso.com"
    }

    record {
    	nsdname = "ns2.contoso.com"
    }

    tags {
	environment = "Production"
	cost_center = "MSFT"
    }
}
`

var testAccAzureRMDnsNsRecord_withTagsUpdate = `
resource "azurerm_resource_group" "test" {
    name = "acctest_rg_%d"
    location = "West US"
}
resource "azurerm_dns_zone" "test" {
    name = "acctestzone%d.com"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_dns_ns_record" "test" {
    name = "myarecord%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    zone_name = "${azurerm_dns_zone.test.name}"
    ttl = "300"
    record {
    	nsdname = "ns1.contoso.com"
    }

    record {
    	nsdname = "ns2.contoso.com"
    }

    tags {
	environment = "staging"
    }
}
`
