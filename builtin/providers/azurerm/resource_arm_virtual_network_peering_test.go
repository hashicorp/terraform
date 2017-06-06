package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMVirtualNetworkPeering_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualNetworkPeering_basic, ri, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualNetworkPeeringDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualNetworkPeeringExists("azurerm_virtual_network_peering.test1"),
					testCheckAzureRMVirtualNetworkPeeringExists("azurerm_virtual_network_peering.test2"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_network_peering.test1", "allow_virtual_network_access", "true"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_network_peering.test2", "allow_virtual_network_access", "true"),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualNetworkPeering_disappears(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualNetworkPeering_basic, ri, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualNetworkPeeringDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualNetworkPeeringExists("azurerm_virtual_network_peering.test1"),
					testCheckAzureRMVirtualNetworkPeeringExists("azurerm_virtual_network_peering.test2"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_network_peering.test1", "allow_virtual_network_access", "true"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_network_peering.test2", "allow_virtual_network_access", "true"),
					testCheckAzureRMVirtualNetworkPeeringDisappears("azurerm_virtual_network_peering.test1"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAzureRMVirtualNetworkPeering_update(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMVirtualNetworkPeering_basic, ri, ri, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMVirtualNetworkPeering_basicUpdate, ri, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualNetworkPeeringDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualNetworkPeeringExists("azurerm_virtual_network_peering.test1"),
					testCheckAzureRMVirtualNetworkPeeringExists("azurerm_virtual_network_peering.test2"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_network_peering.test1", "allow_virtual_network_access", "true"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_network_peering.test2", "allow_virtual_network_access", "true"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_network_peering.test1", "allow_forwarded_traffic", "false"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_network_peering.test2", "allow_forwarded_traffic", "false"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualNetworkPeeringExists("azurerm_virtual_network_peering.test1"),
					testCheckAzureRMVirtualNetworkPeeringExists("azurerm_virtual_network_peering.test2"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_network_peering.test1", "allow_virtual_network_access", "true"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_network_peering.test2", "allow_virtual_network_access", "true"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_network_peering.test1", "allow_forwarded_traffic", "true"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_network_peering.test2", "allow_forwarded_traffic", "true"),
				),
			},
		},
	})
}

func testCheckAzureRMVirtualNetworkPeeringExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		vnetName := rs.Primary.Attributes["virtual_network_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for virtual network peering: %s", name)
		}

		// Ensure resource group/virtual network peering combination exists in API
		conn := testAccProvider.Meta().(*ArmClient).vnetPeeringsClient

		resp, err := conn.Get(resourceGroup, vnetName, name)
		if err != nil {
			return fmt.Errorf("Bad: Get on vnetPeeringsClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Virtual Network Peering %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMVirtualNetworkPeeringDisappears(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		vnetName := rs.Primary.Attributes["virtual_network_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for virtual network peering: %s", name)
		}

		// Ensure resource group/virtual network peering combination exists in API
		conn := testAccProvider.Meta().(*ArmClient).vnetPeeringsClient

		_, error := conn.Delete(resourceGroup, vnetName, name, make(chan struct{}))
		err := <-error
		if err != nil {
			return fmt.Errorf("Bad: Delete on vnetPeeringsClient: %s", err)
		}

		return nil
	}
}

func testCheckAzureRMVirtualNetworkPeeringDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).vnetPeeringsClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_virtual_network_peering" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		vnetName := rs.Primary.Attributes["virtual_network_name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, vnetName, name)
		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Virtual Network Peering sitll exists:\n%#v", resp.VirtualNetworkPeeringPropertiesFormat)
		}
	}

	return nil
}

var testAccAzureRMVirtualNetworkPeering_basic = `
resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "West US"
}

resource "azurerm_virtual_network" "test1" {
  name                = "acctestvirtnet-1-%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  address_space       = ["10.0.1.0/24"]
  location            = "${azurerm_resource_group.test.location}"
}

resource "azurerm_virtual_network" "test2" {
  name                = "acctestvirtnet-2-%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  address_space       = ["10.0.2.0/24"]
  location            = "${azurerm_resource_group.test.location}"
}

resource "azurerm_virtual_network_peering" "test1" {
    name = "acctestpeer-1-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test1.name}"
    remote_virtual_network_id = "${azurerm_virtual_network.test2.id}"
    allow_virtual_network_access = true
}

resource "azurerm_virtual_network_peering" "test2" {
    name = "acctestpeer-2-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test2.name}"
    remote_virtual_network_id = "${azurerm_virtual_network.test1.id}"
    allow_virtual_network_access = true
}
`

var testAccAzureRMVirtualNetworkPeering_basicUpdate = `
resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "West US"
}

resource "azurerm_virtual_network" "test1" {
  name                = "acctestvirtnet-1-%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  address_space       = ["10.0.1.0/24"]
  location            = "${azurerm_resource_group.test.location}"
}

resource "azurerm_virtual_network" "test2" {
  name                = "acctestvirtnet-2-%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  address_space       = ["10.0.2.0/24"]
  location            = "${azurerm_resource_group.test.location}"
}

resource "azurerm_virtual_network_peering" "test1" {
    name = "acctestpeer-1-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test1.name}"
    remote_virtual_network_id = "${azurerm_virtual_network.test2.id}"
    allow_forwarded_traffic = true
    allow_virtual_network_access = true
}

resource "azurerm_virtual_network_peering" "test2" {
    name = "acctestpeer-2-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test2.name}"
    remote_virtual_network_id = "${azurerm_virtual_network.test1.id}"
    allow_forwarded_traffic = true
    allow_virtual_network_access = true
}
`
