package ddcloud

import (
	"fmt"
	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"testing"
)

const testAccDDCloudVLANBasic = `
provider "ddcloud" {
	region		= "AU"
}

resource "ddcloud_networkdomain" "acc_test_domain" {
	name		= "acc-test-domain"
	description	= "Network domain for Terraform acceptance test."
	datacenter	= "AU9"
}

resource "ddcloud_vlan" "test_vlan" {
	name				= "acc-test-vlan"
	description 		= "VLAN for Terraform acceptance test."

	networkdomain 		= "${ddcloud_networkdomain.acc_test_domain.id}"

	ipv4_base_address	= "192.168.17.0"
	ipv4_prefix_size	= 24
}
`

// Acceptance test for ddcloud_vlan (basic):
//
// Create a VLAN and verify that it gets created with the correct configuration.
func TestAccVLANBasicCreate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testCheckDDComputeVLANDestroy,
			testCheckDDComputeNetworkDomainDestroy,
		),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDDCloudVLANBasic,
				Check: resource.ComposeTestCheckFunc(
					testCheckDDCloudVLANExists("ddcloud_networkdomain.acc_test_domain", true),
					testCheckDDCloudVLANMatches("ddcloud_networkdomain.acc_test_domain", compute.VLAN{
						Name:         "acc-test-vlan",
						Description:  "VLAN for Terraform acceptance test.",
						IPv4Range: compute.IPv4Range{
							BaseAddress: "192.168.17.0",
							PrefixSize: 24,
						},
						NetworkDomain: compute.EntitySummary{
							Name: "Network domain for Terraform acceptance test.",
						},
					}),
				),
			},
		},
	})
}

// Acceptance test check for ddcloud_vlan:
//
// Check if the VLAN exists.
func testCheckDDCloudVLANExists(name string, exists bool) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		res, ok := state.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		vlanID := res.Primary.ID

		client := testAccProvider.Meta().(*compute.Client)
		vlan, err := client.GetVLAN(vlanID)
		if err != nil {
			return fmt.Errorf("Bad: Get VLAN: %s", err)
		}
		if exists && vlan == nil {
			return fmt.Errorf("Bad: VLAN not found with Id '%s'.", vlanID)
		} else if !exists && vlan != nil {
			return fmt.Errorf("Bad: VLAN still exists with Id '%s'.", vlanID)
		}

		return nil
	}
}

// Acceptance test check for ddcloud_vlan:
//
// Check if the VLAN's configuration matches the expected configuration.
func testCheckDDCloudVLANMatches(name string, expected compute.VLAN) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		res, ok := state.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		vlanID := res.Primary.ID

		client := testAccProvider.Meta().(*compute.Client)
		vlan, err := client.GetVLAN(vlanID)
		if err != nil {
			return fmt.Errorf("Bad: Get VLAN: %s", err)
		}
		if vlan == nil {
			return fmt.Errorf("Bad: VLAN not found with Id '%s'.", vlanID)
		}

		if vlan.Name != expected.Name {
			return fmt.Errorf("Bad: VLAN '%s' has name '%s' (expected '%s').", vlanID, vlan.Name, expected.Name)
		}

		if vlan.Description != expected.Description {
			return fmt.Errorf("Bad: VLAN '%s' has name '%s' (expected '%s').", vlanID, vlan.Description, expected.Description)
		}

		if vlan.IPv4Range.BaseAddress != expected.IPv4Range.BaseAddress {
			return fmt.Errorf("Bad: VLAN '%s' has IPv4 base address '%s' (expected '%s').", vlanID, vlan.IPv4Range.BaseAddress, expected.IPv4Range.BaseAddress)
		}

		if vlan.IPv4Range.PrefixSize != expected.IPv4Range.PrefixSize {
			return fmt.Errorf("Bad: VLAN '%s' has IPv4 prefix size '%d' (expected '%d').", vlanID, vlan.IPv4Range.PrefixSize, expected.IPv4Range.PrefixSize)
		}

		if vlan.NetworkDomain.Name != expected.NetworkDomain.Name {
			return fmt.Errorf("Bad: VLAN '%s' has network domain named '%s' (expected '%s').", vlanID, vlan.NetworkDomain.Name, expected.NetworkDomain.Name)
		}

		return nil
	}
}

// Acceptance test resource-destruction check for ddcloud_vlan:
//
// Check all VLANs specified in the configuration have been destroyed.
func testCheckDDComputeVLANDestroy(state *terraform.State) error {
	for _, res := range state.RootModule().Resources {
		if res.Type != "ddcloud_vlan" {
			continue
		}

		vlanID := res.Primary.ID

		client := testAccProvider.Meta().(*compute.Client)
		vlan, err := client.GetVLAN(vlanID)
		if err != nil {
			return nil
		}
		if vlan != nil {
			return fmt.Errorf("VLAN '%s' still exists", vlanID)
		}
	}

	return nil
}
