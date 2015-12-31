package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccArmNetworkInterface(t *testing.T) {
	name := "azurerm_virtual_network.test"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAccArmInterfaceDeleted,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureRMNetworkInterfaceConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAccArmInterfaceExists(name),
					resource.TestCheckResourceAttr(name, "name", "acceptanceTestNetworkInterface1"),
					resource.TestCheckResourceAttr(name, "location", "West US"),
					resource.TestCheckResourceAttr(name, "ip_config.0.name", "acceptanceTestIpConfiguration1"),
					resource.TestCheckResourceAttr(name, "ip_config.0.dynamic_private_ip", "true"),
					resource.TestCheckResourceAttr(name, "dns_servers.0", "8.8.8.8"),
					resource.TestCheckResourceAttr(name, "dns_servers.1", "8.8.4.4"),
					resource.TestCheckResourceAttr(name, "applied_servers.0", "8.8.8.8"),
					resource.TestCheckResourceAttr(name, "internal_name", "iface1"),
				),
			},
		},
	})
}

// testCheckAccArmInterfaceExists returns the resource.TestCheckFunc which
// verifies that the virtual network with the provided internal name exists and
// is well defined both within the schema, and on Azure.
func testCheckAccArmInterfaceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// check forexistence in internal state:
		res, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Could not find Network Interface %q.", name)
		}

		resName := res.Primary.Attributes["name"]
		resGrp := res.Primary.Attributes["resource_group_name"]

		ifaceClient := testAccProvider.Meta().(*ArmClient).vnetClient

		resp, err := ifaceClient.Get(resGrp, resName)
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Network interface %q does not exist on Azure!", resName)
		}
		if err != nil {
			return fmt.Errorf("Error reading the state of network interface %q: %s", resName, err)
		}

		return nil
	}
}

// testCheckAccArmInterfaceDeleted is a resource.TestCheckFunc which checks
// that out network interface has been deleted off Azure.
func testCheckAccArmInterfaceDeleted(s *terraform.State) error {
	for _, res := range s.RootModule().Resources {
		if res.Type != "azurerm_network_interface" {
			continue
		}

		name := res.Primary.Attributes["name"]
		resGrp := res.Primary.Attributes["resource_group_name"]

		ifaceClient := testAccProvider.Meta().(ArmClient).ifaceClient
		resp, err := ifaceClient.Get(resGrp, name)

		if resp.StatusCode == http.StatusNotFound {
			return nil
		}

		if err != nil {
			return fmt.Errorf("Error checking if ARM network interface %q got deleted: %s", name, err)
		}
	}

	return nil
}

// testAccAzureRMNetworkInterfaceConfig is the config tests will be conducted upon:
var testAccAzureRMNetworkInterfaceConfig = `
resource "azurerm_resource_group" "test" {
    name = "acceptanceTestResourceGroup1"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    resource_group_name = "${azurerm_resource_group.test.name}"
    name = "acceptanceTestVirtualNetwork1"
    address_space = ["10.0.0.0/16"]
    location = "West US"

    subnet {
        name = "subnet1"
        address_prefix = "10.0.1.0/24"
    }
}

# TODO: resource "azurerm_instance" "test" ...

resource "azurerm_public_ip" "test" {
	name = "testAccPublicIPAddress1"
	resource_group_name = "${azurerm_resource_group.test.name}"
	location = "${azurerm_resource_group.test.name}"
	dns_name = "testAccDnsName1"
	ip_config_id = "${azurerm_network_interface.test.ip_config.0.id}"
}

resource "azurerm_network_interface" "test" {
    resource_group_name = "${azurerm_resource_group.test.name}"
	name = "acceptanceTestPublicIPAddress1"
	location = "West US"
	vm_id = "${azurerm_instance.test.id}"
	# TODO: network_security_group_id = ...

	ip_config = {
		name = "acceptanceTestIpConfiguration1"
		dynamic_private_ip = true
		# TODO: subnet_id = "${azurerm_virtual_network.test.subnet.HASH.id}"
		public_ip_id = "${azurerm_public_ip.test.id}"
	}

	dns_servers = ["8.8.8.8", "8.8.4.4"]
	applied_dns_servers: ["8.8.8.8"]
	internal_name = "iface1"
}
`
