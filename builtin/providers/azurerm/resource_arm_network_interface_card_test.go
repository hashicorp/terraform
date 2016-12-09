package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMNetworkInterface_basic(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMNetworkInterfaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMNetworkInterface_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkInterfaceExists("azurerm_network_interface.test"),
				),
			},
		},
	})
}

func TestAccAzureRMNetworkInterface_disappears(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMNetworkInterfaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMNetworkInterface_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkInterfaceExists("azurerm_network_interface.test"),
					testCheckAzureRMNetworkInterfaceDisappears("azurerm_network_interface.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAzureRMNetworkInterface_enableIPForwarding(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMNetworkInterfaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMNetworkInterface_ipForwarding(rInt),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkInterfaceExists("azurerm_network_interface.test"),
					resource.TestCheckResourceAttr(
						"azurerm_network_interface.test", "enable_ip_forwarding", "true"),
				),
			},
		},
	})
}

func TestAccAzureRMNetworkInterface_withTags(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMNetworkInterfaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMNetworkInterface_withTags(rInt),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkInterfaceExists("azurerm_network_interface.test"),
					resource.TestCheckResourceAttr(
						"azurerm_network_interface.test", "tags.%", "2"),
					resource.TestCheckResourceAttr(
						"azurerm_network_interface.test", "tags.environment", "Production"),
					resource.TestCheckResourceAttr(
						"azurerm_network_interface.test", "tags.cost_center", "MSFT"),
				),
			},
			{
				Config: testAccAzureRMNetworkInterface_withTagsUpdate(rInt),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkInterfaceExists("azurerm_network_interface.test"),
					resource.TestCheckResourceAttr(
						"azurerm_network_interface.test", "tags.%", "1"),
					resource.TestCheckResourceAttr(
						"azurerm_network_interface.test", "tags.environment", "staging"),
				),
			},
		},
	})
}

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

func testCheckAzureRMNetworkInterfaceDisappears(name string) resource.TestCheckFunc {
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

		_, err := conn.Delete(resourceGroup, name, make(chan struct{}))
		if err != nil {
			return fmt.Errorf("Bad: Delete on ifaceClient: %s", err)
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
			return fmt.Errorf("Network Interface still exists:\n%#v", resp.InterfacePropertiesFormat)
		}
	}

	return nil
}

func testAccAzureRMNetworkInterface_basic(rInt int) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctest-rg-%d"
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
`, rInt)
}

func testAccAzureRMNetworkInterface_ipForwarding(rInt int) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctest-rg-%d"
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
    enable_ip_forwarding = true

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}
`, rInt)
}

func testAccAzureRMNetworkInterface_withTags(rInt int) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctest-rg-%d"
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
`, rInt)
}

func testAccAzureRMNetworkInterface_withTagsUpdate(rInt int) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctest-rg-%d"
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
`, rInt)
}
