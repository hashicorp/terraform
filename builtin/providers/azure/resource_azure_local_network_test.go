package azure

import (
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/management"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureLocalNetworkConnectionBasic(t *testing.T) {
	name := "azure_local_network_connection.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccAzureLocalNetworkConnectionDestroyed,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureLocalNetworkConnectionBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccAzureLocalNetworkConnectionExists(name),
					resource.TestCheckResourceAttr(name, "name", "terraform-local-network-connection"),
					resource.TestCheckResourceAttr(name, "vpn_gateway_address", "10.11.12.13"),
					resource.TestCheckResourceAttr(name, "address_space_prefixes.0", "10.10.10.0/31"),
					resource.TestCheckResourceAttr(name, "address_space_prefixes.1", "10.10.10.1/31"),
				),
			},
		},
	})
}

func TestAccAzureLocalNetworkConnectionUpdate(t *testing.T) {
	name := "azure_local_network_connection.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccAzureLocalNetworkConnectionDestroyed,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureLocalNetworkConnectionBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccAzureLocalNetworkConnectionExists(name),
					resource.TestCheckResourceAttr(name, "name", "terraform-local-network-connection"),
					resource.TestCheckResourceAttr(name, "vpn_gateway_address", "10.11.12.13"),
					resource.TestCheckResourceAttr(name, "address_space_prefixes.0", "10.10.10.0/31"),
					resource.TestCheckResourceAttr(name, "address_space_prefixes.1", "10.10.10.1/31"),
				),
			},

			resource.TestStep{
				Config: testAccAzureLocalNetworkConnectionUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccAzureLocalNetworkConnectionExists(name),
					resource.TestCheckResourceAttr(name, "name", "terraform-local-network-connection"),
					resource.TestCheckResourceAttr(name, "vpn_gateway_address", "10.11.12.14"),
					resource.TestCheckResourceAttr(name, "address_space_prefixes.0", "10.10.10.2/30"),
					resource.TestCheckResourceAttr(name, "address_space_prefixes.1", "10.10.10.3/30"),
				),
			},
		},
	})
}

// testAccAzureLocalNetworkConnectionExists checks whether the given local network
// connection exists on Azure.
func testAccAzureLocalNetworkConnectionExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Azure Local Network Connection not found: %s", name)
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("Azure Local Network Connection ID not set.")
		}

		vnetClient := testAccProvider.Meta().(*Client).vnetClient
		netConf, err := vnetClient.GetVirtualNetworkConfiguration()
		if err != nil {
			return err
		}

		for _, lnet := range netConf.Configuration.LocalNetworkSites {
			if lnet.Name == resource.Primary.ID {
				return nil
			}
			break
		}

		return fmt.Errorf("Local Network Connection not found: %s", name)
	}
}

// testAccAzureLocalNetworkConnectionDestroyed checks whether the local network
// connection has been destroyed on Azure or not.
func testAccAzureLocalNetworkConnectionDestroyed(s *terraform.State) error {
	vnetClient := testAccProvider.Meta().(*Client).vnetClient

	for _, resource := range s.RootModule().Resources {
		if resource.Type != "azure_local_network_connection" {
			continue
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("Azure Local Network Connection ID not set.")
		}

		netConf, err := vnetClient.GetVirtualNetworkConfiguration()
		if err != nil {
			// This is desirable - if there is no network config there can be no gateways
			if management.IsResourceNotFoundError(err) {
				continue
			}
			return err
		}

		for _, lnet := range netConf.Configuration.LocalNetworkSites {
			if lnet.Name == resource.Primary.ID {
				return fmt.Errorf("Azure Local Network Connection still exists.")
			}
		}
	}

	return nil
}

const testAccAzureLocalNetworkConnectionBasic = `
resource "azure_local_network_connection" "foo" {
    name = "terraform-local-network-connection"
    vpn_gateway_address = "10.11.12.13"
    address_space_prefixes = ["10.10.10.0/31", "10.10.10.1/31"]
}
`

const testAccAzureLocalNetworkConnectionUpdate = `
resource "azure_local_network_connection" "foo" {
    name = "terraform-local-network-connection"
    vpn_gateway_address = "10.11.12.14"
    address_space_prefixes = ["10.10.10.2/30", "10.10.10.3/30"]
}
`
