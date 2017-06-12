package azurerm

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"net/http"
	"testing"
)

func TestAccAzureRMVirtualNetworkGatewayConnection_sitetosite(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualNetworkGatewayConnection_sitetosite, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualNetworkGatewayConnectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualNetworkGatewayConnectionExists("azurerm_virtual_network_gateway_connection.test"),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualNetworkGatewayConnection_vnettovnet(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualNetworkGatewayConnection_vnettovnet, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualNetworkGatewayConnectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualNetworkGatewayConnectionExists("azurerm_virtual_network_gateway_connection.us_to_europe"),
					testCheckAzureRMVirtualNetworkGatewayConnectionExists("azurerm_virtual_network_gateway_connection.europe_to_us"),
				),
			},
		},
	})
}

func testCheckAzureRMVirtualNetworkGatewayConnectionExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		name, resourceGroup, err := getArmResourceNameAndGroupByTerraformName(s, name)
		if err != nil {
			return err
		}

		conn := testAccProvider.Meta().(*ArmClient).vnetGatewayConnectionsClient

		resp, err := conn.Get(resourceGroup, name)
		if err != nil {
			return fmt.Errorf("Bad: Get on vnetGatewayConnectionsClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Virtual Network Gateway Connection %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMVirtualNetworkGatewayConnectionDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).vnetGatewayConnectionsClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_virtual_network_gateway_connection" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Virtual Network Gateway Connection still exists:\n%#v", resp.VirtualNetworkGatewayConnectionPropertiesFormat)
		}
	}

	return nil
}

var testAccAzureRMVirtualNetworkGatewayConnection_sitetosite = `
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

resource "azurerm_local_network_gateway" "test" {
    name = "test-%[1]d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    gateway_address = "168.62.225.23"
    address_space = ["10.1.1.0/24"]
}

resource "azurerm_virtual_network_gateway_connection" "test" {
    name = "test-%[1]d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    type = "IPsec"
    virtual_network_gateway_id = "${azurerm_virtual_network_gateway.test.id}"
    local_network_gateway_id = "${azurerm_local_network_gateway.test.id}"

    shared_key = "4-v3ry-53cr37-1p53c-5h4r3d-k3y"
}
`

var testAccAzureRMVirtualNetworkGatewayConnection_vnettovnet = `
resource "azurerm_resource_group" "us" {
    name = "us-%[1]d"
    location = "East US"
}

resource "azurerm_virtual_network" "us" {
  name = "us-%[1]d"
  location = "East US"
  resource_group_name = "${azurerm_resource_group.us.name}"
  address_space = ["10.0.0.0/16"]
}

resource "azurerm_subnet" "us_gateway" {
  name = "GatewaySubnet"
  resource_group_name = "${azurerm_resource_group.us.name}"
  virtual_network_name = "${azurerm_virtual_network.us.name}"
  address_prefix = "10.0.1.0/24"
}

resource "azurerm_public_ip" "us" {
  name = "us-%[1]d"
  location = "East US"
  resource_group_name = "${azurerm_resource_group.us.name}"
  public_ip_address_allocation = "Dynamic"
}

resource "azurerm_virtual_network_gateway" "us" {
  name = "us-gateway-%[1]d"
  location = "East US"
  resource_group_name = "${azurerm_resource_group.us.name}"

  type = "Vpn"
  vpn_type = "RouteBased"

	sku {
		name = "Standard"
		tier = "Standard"
	}

  ip_configuration {
    name = "vnetGatewayConfig"
    public_ip_address_id = "${azurerm_public_ip.us.id}"
    private_ip_address_allocation = "Dynamic"
    subnet_id = "${azurerm_subnet.us_gateway.id}"
  }
}

resource "azurerm_virtual_network_gateway_connection" "us_to_europe" {
  name = "us-to-europe-%[1]d"
  location = "East US"
  resource_group_name = "${azurerm_resource_group.us.name}"

  type = "Vnet2Vnet"
  virtual_network_gateway_id = "${azurerm_virtual_network_gateway.us.id}"
  peer_virtual_network_gateway_id = "${azurerm_virtual_network_gateway.europe.id}"

  shared_key = "4-v3ry-53cr37-1p53c-5h4r3d-k3y"
}

resource "azurerm_resource_group" "europe" {
    name = "europe-%[1]d"
    location = "West Europe"
}

resource "azurerm_virtual_network" "europe" {
  name = "europe-%[1]d"
  location = "West Europe"
  resource_group_name = "${azurerm_resource_group.europe.name}"
  address_space = ["10.1.0.0/16"]
}

resource "azurerm_subnet" "europe_gateway" {
  name = "GatewaySubnet"
  resource_group_name = "${azurerm_resource_group.europe.name}"
  virtual_network_name = "${azurerm_virtual_network.europe.name}"
  address_prefix = "10.1.1.0/24"
}

resource "azurerm_public_ip" "europe" {
  name = "europe-%[1]d"
  location = "West Europe"
  resource_group_name = "${azurerm_resource_group.europe.name}"
  public_ip_address_allocation = "Dynamic"
}

resource "azurerm_virtual_network_gateway" "europe" {
  name = "europe-gateway-%[1]d"
  location = "West Europe"
  resource_group_name = "${azurerm_resource_group.europe.name}"

  type = "Vpn"
  vpn_type = "RouteBased"

	sku {
		name = "Standard"
		tier = "Standard"
	}

  ip_configuration {
    name = "vnetGatewayConfig"
    public_ip_address_id = "${azurerm_public_ip.europe.id}"
    private_ip_address_allocation = "Dynamic"
    subnet_id = "${azurerm_subnet.europe_gateway.id}"
  }
}

resource "azurerm_virtual_network_gateway_connection" "europe_to_us" {
  name = "europe-to-us-%[1]d"
  location = "West Europe"
  resource_group_name = "${azurerm_resource_group.europe.name}"

  type = "Vnet2Vnet"
  virtual_network_gateway_id = "${azurerm_virtual_network_gateway.europe.id}"
  peer_virtual_network_gateway_id = "${azurerm_virtual_network_gateway.us.id}"

  shared_key = "4-v3ry-53cr37-1p53c-5h4r3d-k3y"
}
`
