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
	config := fmt.Sprintf(testAccAzureRMVirtualNetworkGateway_basic, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualNetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualNetworkGatewayExists("azurerm_virtual_network_gateway.test"),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualNetworkGateway_withTags(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMVirtualNetworkGateway_withTags, ri, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMVirtualNetworkGateway_withTagsUpdated, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualNetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualNetworkGatewayExists("azurerm_virtual_network_gateway.test"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_network_gateway.test", "tags.%", "2"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_network_gateway.test", "tags.environment", "Production"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_network_gateway.test", "tags.cost_center", "MSFT"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualNetworkGatewayExists("azurerm_virtual_network_gateway.test"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_network_gateway.test", "tags.%", "1"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_network_gateway.test", "tags.environment", "staging"),
				),
			},
		},
	})
}

func testCheckAzureRMVirtualNetworkGatewayExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for virtual network gateway: %s", name)
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
			return fmt.Errorf("Virtual Network Gateway sitll exists:\n%#v", resp.VirtualNetworkGatewayPropertiesFormat)
		}
	}

	return nil
}

var testAccAzureRMVirtualNetworkGateway_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctestvirtnet%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "GatewaySubnet"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.1.0/24"
}

resource "azurerm_public_ip" "test" {
  name                         = "actestpublicip%d"
  location                     = "West US"
  resource_group_name          = "${azurerm_resource_group.test.name}"
  public_ip_address_allocation = "Dynamic"
}

resource "azurerm_virtual_network_gateway" "test" {
    name = "acctestgw-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    enable_bgp = false
    vpn_type = "RouteBased"
    type = "Vpn"

	sku {
		name     = "Basic"
		tier     = "Basic"
	}
    
    ip_configuration {
        name = "vnetGatewayConfig1"
        public_ip_address_id = "${azurerm_public_ip.test.id}"
        private_ip_address_allocation = "Dynamic"
        subnet_id = "${azurerm_subnet.test.id}"
    }
}
`

var testAccAzureRMVirtualNetworkGateway_withTags = `
resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctestvirtnet%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "GatewaySubnet"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.1.0/24"
}

resource "azurerm_public_ip" "test" {
  name                         = "actestpublicip%d"
  location                     = "West US"
  resource_group_name          = "${azurerm_resource_group.test.name}"
  public_ip_address_allocation = "Dynamic"
}

resource "azurerm_virtual_network_gateway" "test" {
    name = "acctestgw-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    enable_bgp = false
    vpn_type = "RouteBased"
    type = "Vpn"

	sku {
		name     = "Basic"
		tier     = "Basic"
	}
    
    ip_configuration {
        name = "vnetGatewayConfig1"
        public_ip_address_id = "${azurerm_public_ip.test.id}"
        private_ip_address_allocation = "Dynamic"
        subnet_id = "${azurerm_subnet.test.id}"
    }

    tags {
        environment = "Production"
        cost_center = "MSFT"
    }
}
`

var testAccAzureRMVirtualNetworkGateway_withTagsUpdated = `
resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctestvirtnet%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "GatewaySubnet"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.1.0/24"
}

resource "azurerm_public_ip" "test" {
  name                         = "actestpublicip%d"
  location                     = "West US"
  resource_group_name          = "${azurerm_resource_group.test.name}"
  public_ip_address_allocation = "Dynamic"
}

resource "azurerm_virtual_network_gateway" "test" {
    name = "acctestgw-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    enable_bgp = false
    vpn_type = "RouteBased"
    type = "Vpn"

	sku {
		name     = "Basic"
		tier     = "Basic"
	}

    ip_configuration {
        name = "vnetGatewayConfig1"
        public_ip_address_id = "${azurerm_public_ip.test.id}"
        private_ip_address_allocation = "Dynamic"
        subnet_id = "${azurerm_subnet.test.id}"
    }

    tags {
        environment = "staging"
    }
}
`
