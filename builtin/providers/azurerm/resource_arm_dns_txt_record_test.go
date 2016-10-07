package azurerm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/jen20/riviera/dns"
)

func TestAccAzureRMDnsTxtRecord_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMDnsTxtRecord_basic, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMDnsTxtRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDnsTxtRecordExists("azurerm_dns_txt_record.test"),
				),
			},
		},
	})
}

func TestAccAzureRMDnsTxtRecord_updateRecords(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMDnsTxtRecord_basic, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMDnsTxtRecord_updateRecords, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMDnsTxtRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDnsTxtRecordExists("azurerm_dns_txt_record.test"),
					resource.TestCheckResourceAttr(
						"azurerm_dns_txt_record.test", "record.#", "2"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDnsTxtRecordExists("azurerm_dns_txt_record.test"),
					resource.TestCheckResourceAttr(
						"azurerm_dns_txt_record.test", "record.#", "3"),
				),
			},
		},
	})
}

func TestAccAzureRMDnsTxtRecord_withTags(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMDnsTxtRecord_withTags, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMDnsTxtRecord_withTagsUpdate, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMDnsTxtRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDnsTxtRecordExists("azurerm_dns_txt_record.test"),
					resource.TestCheckResourceAttr(
						"azurerm_dns_txt_record.test", "tags.%", "2"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDnsTxtRecordExists("azurerm_dns_txt_record.test"),
					resource.TestCheckResourceAttr(
						"azurerm_dns_txt_record.test", "tags.%", "1"),
				),
			},
		},
	})
}

func testCheckAzureRMDnsTxtRecordExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).rivieraClient

		readRequest := conn.NewRequestForURI(rs.Primary.ID)
		readRequest.Command = &dns.GetTXTRecordSet{}

		readResponse, err := readRequest.Execute()
		if err != nil {
			return fmt.Errorf("Bad: GetTXTRecordSet: %s", err)
		}
		if !readResponse.IsSuccessful() {
			return fmt.Errorf("Bad: GetTXTRecordSet: %s", readResponse.Error)
		}

		return nil
	}
}

func testCheckAzureRMDnsTxtRecordDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).rivieraClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_dns_txt_record" {
			continue
		}

		readRequest := conn.NewRequestForURI(rs.Primary.ID)
		readRequest.Command = &dns.GetTXTRecordSet{}

		readResponse, err := readRequest.Execute()
		if err != nil {
			return fmt.Errorf("Bad: GetTXTRecordSet: %s", err)
		}

		if readResponse.IsSuccessful() {
			return fmt.Errorf("Bad: DNS TXT Record still exists: %s", readResponse.Error)
		}
	}

	return nil
}

var testAccAzureRMDnsTxtRecord_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG_%d"
    location = "West US"
}
resource "azurerm_dns_zone" "test" {
    name = "acctestzone%d.com"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_dns_txt_record" "test" {
    name = "myarecord%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    zone_name = "${azurerm_dns_zone.test.name}"
    ttl = "300"

    record {
    	value = "Quick brown fox"
    }

    record {
    	value = "Another test txt string"
    }
}
`

var testAccAzureRMDnsTxtRecord_updateRecords = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG_%d"
    location = "West US"
}
resource "azurerm_dns_zone" "test" {
    name = "acctestzone%d.com"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_dns_txt_record" "test" {
    name = "myarecord%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    zone_name = "${azurerm_dns_zone.test.name}"
    ttl = "300"

    record {
    	value = "Quick brown fox"
    }

    record {
    	value = "Another test txt string"
    }

    record {
    	value = "A wild 3rd record appears"
    }
}
`

var testAccAzureRMDnsTxtRecord_withTags = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG_%d"
    location = "West US"
}
resource "azurerm_dns_zone" "test" {
    name = "acctestzone%d.com"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_dns_txt_record" "test" {
    name = "myarecord%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    zone_name = "${azurerm_dns_zone.test.name}"
    ttl = "300"

    record {
    	value = "Quick brown fox"
    }

    record {
    	value = "Another test txt string"
    }

    tags {
	environment = "Production"
	cost_center = "MSFT"
    }
}
`

var testAccAzureRMDnsTxtRecord_withTagsUpdate = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG_%d"
    location = "West US"
}
resource "azurerm_dns_zone" "test" {
    name = "acctestzone%d.com"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_dns_txt_record" "test" {
    name = "myarecord%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    zone_name = "${azurerm_dns_zone.test.name}"
    ttl = "300"
    record {
    	value = "Quick brown fox"
    }

    record {
    	value = "Another test txt string"
    }

    tags {
	environment = "staging"
    }
}
`
