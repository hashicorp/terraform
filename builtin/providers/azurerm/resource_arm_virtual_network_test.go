package azurerm

import (
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/core/http"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMVirtualNetwork_basic(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualNetwork_basic, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualNetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualNetworkExists("azurerm_virtual_network.test"),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualNetwork_withTags(t *testing.T) {

	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMVirtualNetwork_withTags, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMVirtualNetwork_withTagsUpdated, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualNetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualNetworkExists("azurerm_virtual_network.test"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_network.test", "tags.#", "2"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_network.test", "tags.environment", "Production"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_network.test", "tags.cost_center", "MSFT"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualNetworkExists("azurerm_virtual_network.test"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_network.test", "tags.#", "1"),
					resource.TestCheckResourceAttr(
						"azurerm_virtual_network.test", "tags.environment", "staging"),
				),
			},
		},
	})
}

func testCheckAzureRMVirtualNetworkExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		virtualNetworkName := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for virtual network: %s", virtualNetworkName)
		}

		// Ensure resource group/virtual network combination exists in API
		conn := testAccProvider.Meta().(*ArmClient).vnetClient

		resp, err := conn.Get(resourceGroup, virtualNetworkName, "")
		if err != nil {
			return fmt.Errorf("Bad: Get on vnetClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Virtual Network %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMVirtualNetworkDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).vnetClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_virtual_network" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name, "")

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Virtual Network sitll exists:\n%#v", resp.Properties)
		}
	}

	return nil
}

var testAccAzureRMVirtualNetwork_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctestvirtnet%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    subnet {
        name = "subnet1"
        address_prefix = "10.0.1.0/24"
    }
}
`

var testAccAzureRMVirtualNetwork_withTags = `
resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctestvirtnet%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    subnet {
        name = "subnet1"
        address_prefix = "10.0.1.0/24"
    }

    tags {
	environment = "Production"
	cost_center = "MSFT"
    }
}
`

var testAccAzureRMVirtualNetwork_withTagsUpdated = `
resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctestvirtnet%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    subnet {
        name = "subnet1"
        address_prefix = "10.0.1.0/24"
    }

    tags {
	environment = "staging"
    }
}
`
