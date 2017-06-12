package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMVirtualNetworkGateway_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualNetworkGateway_basic, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualNetworkGatewayDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualNetworkGatewayExists("azurerm_virtual_network_gateway.test"),
				),
			},
		},
	})
}

func testCheckAzureRMVirtualNetworkGatewayExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		name, resourceGroup, err := getArmResourceNameAndGroupByTerraformName(s, name)
		if err != nil {
			return err
		}

		conn := testAccProvider.Meta().(*ArmClient).vnetGatewayClient

		resp, err := conn.Get(resourceGroup, name)
		if err != nil {
			return fmt.Errorf("Bad: Get on vnetGatewayClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Virtual Network Gateway %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMVirtualNetworkGatewayDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).vnetGatewayClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_virtual_network_gateway" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Virtual Network Gateway still exists:\n%#v", resp.VirtualNetworkGatewayPropertiesFormat)
		}
	}

	return nil
}

var testAccAzureRMVirtualNetworkGateway_basic = `
resource "azurerm_resource_group" "test" {
    name = "test-%[1]d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
  name = "test-%[1]d"
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  address_space = ["10.0.0.0/16"]
}

resource "azurerm_subnet" "test" {
  name = "GatewaySubnet"
  resource_group_name = "${azurerm_resource_group.test.name}"
  virtual_network_name = "${azurerm_virtual_network.test.name}"
  address_prefix = "10.0.1.0/24"
}

resource "azurerm_public_ip" "test" {
    name = "test-%[1]d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    public_ip_address_allocation = "Dynamic"
}

resource "azurerm_virtual_network_gateway" "test" {
  name = "test-%[1]d"
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"

  type = "Vpn"
  vpn_type = "RouteBased"

  sku {
    name = "Basic"
    tier = "Basic"
  }

  ip_configuration {
    name = "vnetGatewayConfig"
    public_ip_address_id = "${azurerm_public_ip.test.id}"
    private_ip_address_allocation = "Dynamic"
    subnet_id = "${azurerm_subnet.test.id}"
  }
}
`
