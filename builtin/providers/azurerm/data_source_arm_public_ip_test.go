package azurerm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceAzureRMPublicIP_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := testAccDatSourceAzureRMPublicIPBasic(ri)
	name := fmt.Sprintf("acctestpublicip-%d", ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMPublicIpDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.azurerm_public_ip.test", "name", name),
					resource.TestCheckResourceAttr("data.azurerm_public_ip.test", "location", "westus"),
					resource.TestCheckResourceAttr("data.azurerm_public_ip.test", "tags.%", "1"),
					resource.TestCheckResourceAttr("data.azurerm_public_ip.test", "tags.env", "test"),
				),
			},
		},
	})
}

func testAccDatSourceAzureRMPublicIPBasic(ri int) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}
resource "azurerm_public_ip" "test" {
    name = "acctestpublicip-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    public_ip_address_allocation = "static"

    tags {
	environment = "test"
    }
}

data "azurerm_public_ip" "test" {
    name = "acctestpublicip-%d"
    resource_group_name = "acctestRG-%d"
}
`, ri, ri, ri, ri)
}
