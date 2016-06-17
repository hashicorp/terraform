package ddcloud

import (
	"fmt"
	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"testing"
)

const testAccDDCloudNetworkDomainBasic = `
provider "ddcloud" {
	region		= "AU"
}

resource "ddcloud_networkdomain" "acc_test_domain" {
	name		= "acc-test-domain"
	description	= "Network domain for Terraform acceptance test."
	datacenter	= "AU9"
}
`

// Acceptance test for ddcloud_networkdomain (basic):
//
// Create a network domain and verify that it gets created with the correct configuration.
func TestAccNetworkDomainBasicCreate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testCheckDDComputeNetworkDomainDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDDCloudNetworkDomainBasic,
				Check: resource.ComposeTestCheckFunc(
					testCheckDDCloudNetworkDomainExists("ddcloud_networkdomain.acc_test_domain", true),
					testCheckDDCloudNetworkDomainMatches("ddcloud_networkdomain.acc_test_domain", compute.NetworkDomain{
						Name:         "acc-test-domain",
						Description:  "Network domain for Terraform acceptance test.",
						DatacenterID: "AU9",
					}),
				),
			},
		},
	})
}

// Acceptance test check for ddcloud_networkdomain:
//
// Check if the network domain exists.
func testCheckDDCloudNetworkDomainExists(name string, exists bool) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		res, ok := state.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		networkDomainID := res.Primary.ID

		client := testAccProvider.Meta().(*compute.Client)
		networkDomain, err := client.GetNetworkDomain(networkDomainID)
		if err != nil {
			return fmt.Errorf("Bad: Get network domain: %s", err)
		}
		if exists && networkDomain == nil {
			return fmt.Errorf("Bad: Network domain not found with Id '%s'.", networkDomainID)
		} else if !exists && networkDomain != nil {
			return fmt.Errorf("Bad: Network domain still exists with Id '%s'.", networkDomainID)
		}

		return nil
	}
}

// Acceptance test check for ddcloud_networkdomain:
//
// Check if the network domain's configuration matches the expected configuration.
func testCheckDDCloudNetworkDomainMatches(name string, expected compute.NetworkDomain) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		res, ok := state.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		networkDomainID := res.Primary.ID

		client := testAccProvider.Meta().(*compute.Client)
		networkDomain, err := client.GetNetworkDomain(networkDomainID)
		if err != nil {
			return fmt.Errorf("Bad: Get network domain: %s", err)
		}
		if networkDomain == nil {
			return fmt.Errorf("Bad: Network domain not found with Id '%s'.", networkDomainID)
		}

		if networkDomain.Name != expected.Name {
			return fmt.Errorf("Bad: Network domain '%s' has name '%s' (expected '%s').", networkDomainID, networkDomain.Name, expected.Name)
		}

		if networkDomain.Description != expected.Description {
			return fmt.Errorf("Bad: Network domain '%s' has name '%s' (expected '%s').", networkDomainID, networkDomain.Description, expected.Description)
		}

		return nil
	}
}

// Acceptance test check for ddcloud_networkdomain:
//
// Check all network domains specified in the configuration have been destroyed.
func testCheckDDComputeNetworkDomainDestroy(state *terraform.State) error {
	for _, res := range state.RootModule().Resources {
		if res.Type != "ddcloud_networkdomain" {
			continue
		}

		networkDomainID := res.Primary.ID

		client := testAccProvider.Meta().(*compute.Client)
		networkDomain, err := client.GetNetworkDomain(networkDomainID)
		if err != nil {
			return nil
		}
		if networkDomain != nil {
			return fmt.Errorf("Network domain '%s' still exists.", networkDomainID)
		}
	}

	return nil
}
