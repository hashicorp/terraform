package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMNetworkInterface_basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMNetworkInterfaceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureRMNetworkInterface_basic,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkInterfaceExists("azurerm_network_interface.test"),
				),
			},
		},
	})
}

func TestAccAzureRMNetworkInterface_withTags(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMNetworkInterfaceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureRMNetworkInterface_withTags,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkInterfaceExists("azurerm_network_interface.test"),
					resource.TestCheckResourceAttr(
						"azurerm_network_interface.test", "tags.#", "2"),
					resource.TestCheckResourceAttr(
						"azurerm_network_interface.test", "tags.environment", "Production"),
					resource.TestCheckResourceAttr(
						"azurerm_network_interface.test", "tags.cost_center", "MSFT"),
				),
			},

			resource.TestStep{
				Config: testAccAzureRMNetworkInterface_withTagsUpdate,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkInterfaceExists("azurerm_network_interface.test"),
					resource.TestCheckResourceAttr(
						"azurerm_network_interface.test", "tags.#", "1"),
					resource.TestCheckResourceAttr(
						"azurerm_network_interface.test", "tags.environment", "staging"),
				),
			},
		},
	})
}

///TODO: Re-enable this test when https://github.com/Azure/azure-sdk-for-go/issues/259 is fixed
//func TestAccAzureRMNetworkInterface_addingIpConfigurations(t *testing.T) {
//
//	resource.Test(t, resource.TestCase{
//		PreCheck:     func() { testAccPreCheck(t) },
//		Providers:    testAccProviders,
//		CheckDestroy: testCheckAzureRMNetworkInterfaceDestroy,
//		Steps: []resource.TestStep{
//			resource.TestStep{
//				Config: testAccAzureRMNetworkInterface_basic,
//				Check: resource.ComposeTestCheckFunc(
//					testCheckAzureRMNetworkInterfaceExists("azurerm_network_interface.test"),
//					resource.TestCheckResourceAttr(
//						"azurerm_network_interface.test", "ip_configuration.#", "1"),
//				),
//			},
//
//			resource.TestStep{
//				Config: testAccAzureRMNetworkInterface_extraIpConfiguration,
//				Check: resource.ComposeTestCheckFunc(
//					testCheckAzureRMNetworkInterfaceExists("azurerm_network_interface.test"),
//					resource.TestCheckResourceAttr(
//						"azurerm_network_interface.test", "ip_configuration.#", "2"),
//				),
//			},
//		},
//	})
//}

func testCheckAzureRMNetworkInterfaceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for availability set: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).ifaceClient

		resp, err := conn.Get(resourceGroup, name, "")
		if err != nil {
			return fmt.Errorf("Bad: Get on ifaceClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Network Interface %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMNetworkInterfaceDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).ifaceClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_network_interface" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name, "")

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Network Interface still exists:\n%#v", resp.Properties)
		}
	}

	return nil
}

var testAccAzureRMNetworkInterface_basic = `
resource "azurerm_resource_group" "test" {
    name = "acceptanceTestResourceGroup1"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acceptanceTestVirtualNetwork1"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "testsubnet"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acceptanceTestNetworkInterface1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}
`

var testAccAzureRMNetworkInterface_withTags = `
resource "azurerm_resource_group" "test" {
    name = "acceptanceTestResourceGroup1"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acceptanceTestVirtualNetwork1"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "testsubnet"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acceptanceTestNetworkInterface1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }

    tags {
	environment = "Production"
	cost_center = "MSFT"
    }
}
`

var testAccAzureRMNetworkInterface_withTagsUpdate = `
resource "azurerm_resource_group" "test" {
    name = "acceptanceTestResourceGroup1"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acceptanceTestVirtualNetwork1"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "testsubnet"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acceptanceTestNetworkInterface1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }

    tags {
	environment = "staging"
    }
}
`

//TODO: Re-enable this test when https://github.com/Azure/azure-sdk-for-go/issues/259 is fixed
//var testAccAzureRMNetworkInterface_extraIpConfiguration = `
//resource "azurerm_resource_group" "test" {
//    name = "acceptanceTestResourceGroup1"
//    location = "West US"
//}
//
//resource "azurerm_virtual_network" "test" {
//    name = "acceptanceTestVirtualNetwork1"
//    address_space = ["10.0.0.0/16"]
//    location = "West US"
//    resource_group_name = "${azurerm_resource_group.test.name}"
//}
//
//resource "azurerm_subnet" "test" {
//    name = "testsubnet"
//    resource_group_name = "${azurerm_resource_group.test.name}"
//    virtual_network_name = "${azurerm_virtual_network.test.name}"
//    address_prefix = "10.0.2.0/24"
//}
//
//resource "azurerm_subnet" "test1" {
//    name = "testsubnet1"
//    resource_group_name = "${azurerm_resource_group.test.name}"
//    virtual_network_name = "${azurerm_virtual_network.test.name}"
//    address_prefix = "10.0.1.0/24"
//}
//
//resource "azurerm_network_interface" "test" {
//    name = "acceptanceTestNetworkInterface1"
//    location = "West US"
//    resource_group_name = "${azurerm_resource_group.test.name}"
//
//    ip_configuration {
//    	name = "testconfiguration1"
//    	subnet_id = "${azurerm_subnet.test.id}"
//    	private_ip_address_allocation = "dynamic"
//    }
//
//    ip_configuration {
//    	name = "testconfiguration2"
//    	subnet_id = "${azurerm_subnet.test1.id}"
//    	private_ip_address_allocation = "dynamic"
//    	primary = true
//    }
//}
//`
