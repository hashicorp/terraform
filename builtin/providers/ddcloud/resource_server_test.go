package ddcloud

import (
	"fmt"
	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"testing"
)

/*
 * Acceptance-test configurations.
 */

// A basic Server (and its accompanying network domain and VLAN).
func testAccDDCloudServerBasic(name string, description string, primaryIPv4Address string) string {
	return fmt.Sprintf(`
		provider "ddcloud" {
			region		= "AU"
		}

		resource "ddcloud_networkdomain" "acc_test_domain" {
			name		= "acc-test-networkdomain"
			description	= "Network domain for Terraform acceptance test."
			datacenter	= "AU9"
		}

		resource "ddcloud_vlan" "acc_test_vlan" {
			name				= "acc-test-vlan"
			description 		= "VLAN for Terraform acceptance test."

			networkdomain 		= "${ddcloud_networkdomain.acc_test_domain.id}"

			ipv4_base_address	= "192.168.17.0"
			ipv4_prefix_size	= 24
		}

		resource "ddcloud_server" "acc_test_server" {
			name				 = "%s"
			description 		 = "%s"
			admin_password		 = "snausages!"

			memory_gb			 = 8

			networkdomain 		 = "${ddcloud_networkdomain.acc_test_domain.id}"
			primary_adapter_ipv4 = "%s"
			dns_primary			 = "8.8.8.8"
			dns_secondary		 = "8.8.4.4"

			osimage_name		 = "CentOS 7 64-bit 2 CPU"

			auto_start			 = true

			depends_on			 = ["ddcloud_vlan.acc_test_vlan"]
		}
	`, name, description, primaryIPv4Address)
}

/*
 * Acceptance tests.
 */

// Acceptance test for ddcloud_server (basic):
//
// Create a server and verify that it gets created with the correct configuration.
func TestAccServerBasicCreate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testCheckDDCloudServerDestroy,
			testCheckDDCloudVLANDestroy,
			testCheckDDCloudNetworkDomainDestroy,
		),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDDCloudServerBasic(
					"acc-test-server",
					"Server for Terraform acceptance test.",
					"192.168.17.4",
				),
				Check: resource.ComposeTestCheckFunc(
					testCheckDDCloudServerExists("ddcloud_server.acc_test_server", true),
					testCheckDDCloudServerMatches(
						"ddcloud_server.acc_test_server",
						"ddcloud_networkdomain.acc_test_domain",
						compute.Server{
							Name:        "acc-test-server",
							Description: "Server for Terraform acceptance test.",
							MemoryGB:    8,
							Network: compute.VirtualMachineNetwork{
								PrimaryAdapter: compute.VirtualMachineNetworkAdapter{
									PrivateIPv4Address: stringToPtr("192.168.17.4"),
								},
							},
						},
					),
				),
			},
		},
	})
}

/*
 * Acceptance-test checks.
 */

// Acceptance test check for ddcloud_server:
//
// Check if the server exists.
func testCheckDDCloudServerExists(name string, exists bool) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		res, ok := state.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		serverID := res.Primary.ID

		client := testAccProvider.Meta().(*compute.Client)
		server, err := client.GetServer(serverID)
		if err != nil {
			return fmt.Errorf("Bad: Get server: %s", err)
		}
		if exists && server == nil {
			return fmt.Errorf("Bad: Server not found with Id '%s'.", serverID)
		} else if !exists && server != nil {
			return fmt.Errorf("Bad: Server still exists with Id '%s'.", serverID)
		}

		return nil
	}
}

// Acceptance test check for ddcloud_server:
//
// Check if the server's configuration matches the expected configuration.
func testCheckDDCloudServerMatches(name string, networkDomainName string, expected compute.Server) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		serverResource, ok := state.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		serverID := serverResource.Primary.ID

		client := testAccProvider.Meta().(*compute.Client)
		server, err := client.GetServer(serverID)
		if err != nil {
			return fmt.Errorf("Bad: Get server: %s", err)
		}
		if server == nil {
			return fmt.Errorf("Bad: Server not found with Id '%s'", serverID)
		}

		if server.Name != expected.Name {
			return fmt.Errorf("Bad: Server '%s' has name '%s' (expected '%s')", serverID, server.Name, expected.Name)
		}

		if server.Description != expected.Description {
			return fmt.Errorf("Bad: Server '%s' has name '%s' (expected '%s')", serverID, server.Description, expected.Description)
		}

		if server.MemoryGB != expected.MemoryGB {
			return fmt.Errorf("Bad: Server '%s' has been allocated %dGB of memory (expected %dGB)", serverID, server.MemoryGB, expected.MemoryGB)
		}

		expectedPrimaryIPv4 := *expected.Network.PrimaryAdapter.PrivateIPv4Address
		actualPrimaryIPv4 := *server.Network.PrimaryAdapter.PrivateIPv4Address
		if actualPrimaryIPv4 != expectedPrimaryIPv4 {
			return fmt.Errorf("Bad: Primary network adapter for server '%s' has IPv4 address '%s' (expected '%s')", serverID, actualPrimaryIPv4, expectedPrimaryIPv4)
		}

		expectedPrimaryIPv6, ok := serverResource.Primary.Attributes[resourceKeyServerPrimaryIPv6]
		if !ok {
			return fmt.Errorf("Bad: %s.%s is missing '%s' attribute.", serverResource.Type, name, resourceKeyServerPrimaryIPv6)
		}

		actualPrimaryIPv6 := *server.Network.PrimaryAdapter.PrivateIPv6Address
		if actualPrimaryIPv6 != expectedPrimaryIPv6 {
			return fmt.Errorf("Bad: Primary network adapter for server '%s' has IPv6 address '%s' (expected '%s')", serverID, actualPrimaryIPv6, expectedPrimaryIPv6)
		}

		networkDomainResource := state.RootModule().Resources[networkDomainName]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		expectedNetworkDomainID := networkDomainResource.Primary.ID
		if server.Network.NetworkDomainID != expectedNetworkDomainID {
			return fmt.Errorf("Bad: Server '%s' is part of network domain '%s' (expected '%s')", serverID, server.Network.NetworkDomainID, expectedNetworkDomainID)
		}

		return nil
	}
}

// Acceptance test resource-destruction check for ddcloud_server:
//
// Check all servers specified in the configuration have been destroyed.
func testCheckDDCloudServerDestroy(state *terraform.State) error {
	for _, res := range state.RootModule().Resources {
		if res.Type != "ddcloud_server" {
			continue
		}

		serverID := res.Primary.ID

		client := testAccProvider.Meta().(*compute.Client)
		server, err := client.GetServer(serverID)
		if err != nil {
			return nil
		}
		if server != nil {
			return fmt.Errorf("Server '%s' still exists", serverID)
		}
	}

	return nil
}
