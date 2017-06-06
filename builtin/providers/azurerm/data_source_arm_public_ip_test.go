package azurerm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceAzureRMPublicIP_basic(t *testing.T) {
	ri := acctest.RandInt()

	name := fmt.Sprintf("acctestpublicip-%d", ri)
	resourceGroupName := fmt.Sprintf("acctestRG-%d", ri)

	config := testAccDatSourceAzureRMPublicIPBasic(name, resourceGroupName)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMPublicIpDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.azurerm_public_ip.test", "name", name),
					resource.TestCheckResourceAttr("data.azurerm_public_ip.test", "resource_group_name", resourceGroupName),
					resource.TestCheckResourceAttr("data.azurerm_public_ip.test", "domain_name_label", "mylabel01"),
					resource.TestCheckResourceAttr("data.azurerm_public_ip.test", "idle_timeout_in_minutes", "30"),
					resource.TestCheckResourceAttrSet("data.azurerm_public_ip.test", "fqdn"),
					resource.TestCheckResourceAttrSet("data.azurerm_public_ip.test", "ip_address"),
					resource.TestCheckResourceAttr("data.azurerm_public_ip.test", "tags.%", "1"),
					resource.TestCheckResourceAttr("data.azurerm_public_ip.test", "tags.environment", "test"),
				),
			},
		},
	})
}

func testAccDatSourceAzureRMPublicIPBasic(name string, resourceGroupName string) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "%s"
    location = "West US"
}
resource "azurerm_public_ip" "test" {
    name = "%s"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    public_ip_address_allocation = "static"
	domain_name_label = "mylabel01"
	idle_timeout_in_minutes = 30
	
    tags {
		environment = "test"
    }
}

data "azurerm_public_ip" "test" {
    name = "${azurerm_public_ip.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
}
`, resourceGroupName, name)
}
