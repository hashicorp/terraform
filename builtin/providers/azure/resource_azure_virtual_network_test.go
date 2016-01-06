package azure

import (
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/management"
	"github.com/Azure/azure-sdk-for-go/management/virtualnetwork"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureVirtualNetwork_basic(t *testing.T) {
	var network virtualnetwork.VirtualNetworkSite

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureVirtualNetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureVirtualNetwork_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureVirtualNetworkExists(
						"azure_virtual_network.foo", &network),
					testAccCheckAzureVirtualNetworkAttributes(&network),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "name", "terraform-vnet"),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "location", "West US"),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "address_space.0", "10.1.2.0/24"),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "subnet.1787288781.name", "subnet1"),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "subnet.1787288781.address_prefix", "10.1.2.0/25"),
				),
			},
		},
	})
}

func TestAccAzureVirtualNetwork_advanced(t *testing.T) {
	var network virtualnetwork.VirtualNetworkSite

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureVirtualNetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureVirtualNetwork_advanced,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureVirtualNetworkExists(
						"azure_virtual_network.foo", &network),
					testAccCheckAzureVirtualNetworkAttributes(&network),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "name", "terraform-vnet"),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "location", "West US"),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "address_space.0", "10.1.2.0/24"),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "subnet.33778499.name", "subnet1"),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "subnet.33778499.address_prefix", "10.1.2.0/25"),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "subnet.33778499.security_group", "terraform-security-group1"),
				),
			},
		},
	})
}

func TestAccAzureVirtualNetwork_update(t *testing.T) {
	var network virtualnetwork.VirtualNetworkSite

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureVirtualNetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureVirtualNetwork_advanced,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureVirtualNetworkExists(
						"azure_virtual_network.foo", &network),
					testAccCheckAzureVirtualNetworkAttributes(&network),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "name", "terraform-vnet"),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "location", "West US"),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "address_space.0", "10.1.2.0/24"),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "subnet.33778499.name", "subnet1"),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "subnet.33778499.address_prefix", "10.1.2.0/25"),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "subnet.33778499.security_group", "terraform-security-group1"),
				),
			},

			resource.TestStep{
				Config: testAccAzureVirtualNetwork_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureVirtualNetworkExists(
						"azure_virtual_network.foo", &network),
					testAccCheckAzureVirtualNetworkAttributes(&network),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "name", "terraform-vnet"),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "location", "West US"),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "address_space.0", "10.1.3.0/24"),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "subnet.514595123.name", "subnet1"),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "subnet.514595123.address_prefix", "10.1.3.128/25"),
					resource.TestCheckResourceAttr(
						"azure_virtual_network.foo", "subnet.514595123.security_group", "terraform-security-group2"),
				),
			},
		},
	})
}

func testAccCheckAzureVirtualNetworkExists(
	n string,
	network *virtualnetwork.VirtualNetworkSite) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Virtual Network ID is set")
		}

		vnetClient := testAccProvider.Meta().(*Client).vnetClient
		nc, err := vnetClient.GetVirtualNetworkConfiguration()
		if err != nil {
			return err
		}

		for _, n := range nc.Configuration.VirtualNetworkSites {
			if n.Name == rs.Primary.ID {
				*network = n

				return nil
			}
		}

		return fmt.Errorf("Virtual Network not found")
	}
}

func testAccCheckAzureVirtualNetworkAttributes(
	network *virtualnetwork.VirtualNetworkSite) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if network.Name != "terraform-vnet" {
			return fmt.Errorf("Bad name: %s", network.Name)
		}

		if network.Location != "West US" {
			return fmt.Errorf("Bad location: %s", network.Location)
		}

		return nil
	}
}

func testAccCheckAzureVirtualNetworkDestroy(s *terraform.State) error {
	vnetClient := testAccProvider.Meta().(*Client).vnetClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azure_virtual_network" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Virtual Network ID is set")
		}

		nc, err := vnetClient.GetVirtualNetworkConfiguration()
		if err != nil {
			if management.IsResourceNotFoundError(err) {
				// This is desirable - no configuration = no networks
				continue
			}
			return fmt.Errorf("Error retrieving Virtual Network Configuration: %s", err)
		}

		for _, n := range nc.Configuration.VirtualNetworkSites {
			if n.Name == rs.Primary.ID {
				return fmt.Errorf("Virtual Network %s still exists", rs.Primary.ID)
			}
		}
	}

	return nil
}

const testAccAzureVirtualNetwork_basic = `
resource "azure_virtual_network" "foo" {
    name = "terraform-vnet"
    address_space = ["10.1.2.0/24"]
    location = "West US"

    subnet {
        name = "subnet1"
        address_prefix = "10.1.2.0/25"
    }
}`

const testAccAzureVirtualNetwork_advanced = `
resource "azure_security_group" "foo" {
    name = "terraform-security-group1"
    location = "West US"
}

resource "azure_security_group_rule" "foo" {
	name = "terraform-secgroup-rule"
	security_group_names = ["${azure_security_group.foo.name}"]
	type = "Inbound"
	action = "Deny"
	priority = 200
	source_address_prefix = "100.0.0.0/32"
	source_port_range = "1000"
	destination_address_prefix = "10.0.0.0/32"
	destination_port_range = "1000"
	protocol = "TCP"
}

resource "azure_virtual_network" "foo" {
    name = "terraform-vnet"
    address_space = ["10.1.2.0/24"]
    location = "West US"

    subnet {
        name = "subnet1"
        address_prefix = "10.1.2.0/25"
        security_group = "${azure_security_group.foo.name}"
    }
}`

const testAccAzureVirtualNetwork_update = `
resource "azure_security_group" "foo" {
    name = "terraform-security-group1"
    location = "West US"
}

resource "azure_security_group_rule" "foo" {
	name = "terraform-secgroup-rule"
	security_group_names = ["${azure_security_group.foo.name}"]
	type = "Inbound"
	action = "Deny"
	priority = 200
	source_address_prefix = "100.0.0.0/32"
	source_port_range = "1000"
	destination_address_prefix = "10.0.0.0/32"
	destination_port_range = "1000"
	protocol = "TCP"
}

resource "azure_security_group" "bar" {
    name = "terraform-security-group2"
    location = "West US"
}

resource "azure_virtual_network" "foo" {
    name = "terraform-vnet"
    address_space = ["10.1.3.0/24"]
    location = "West US"

    subnet {
        name = "subnet1"
        address_prefix = "10.1.3.128/25"
        security_group = "${azure_security_group.bar.name}"
    }
}`
