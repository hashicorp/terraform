package openstack

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/fwaas/firewalls"
)

func TestAccFWFirewallV1_basic(t *testing.T) {

	var policyID *string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckFWFirewallV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testFirewallConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFWFirewallV1Exists("openstack_fw_firewall_v1.accept_test", "", "", policyID),
				),
			},
			resource.TestStep{
				Config: testFirewallConfigUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFWFirewallV1Exists("openstack_fw_firewall_v1.accept_test", "accept_test", "terraform acceptance test", policyID),
				),
			},
		},
	})
}

func testAccCheckFWFirewallV1Destroy(s *terraform.State) error {

	config := testAccProvider.Meta().(*Config)
	networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckOpenstackFirewallDestroy) Error creating OpenStack networking client: %s", err)
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_firewall" {
			continue
		}
		_, err = firewalls.Get(networkingClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Firewall (%s) still exists.", rs.Primary.ID)
		}
		httpError, ok := err.(*gophercloud.UnexpectedResponseCodeError)
		if !ok || httpError.Actual != 404 {
			return httpError
		}
	}
	return nil
}

func testAccCheckFWFirewallV1Exists(n, expectedName, expectedDescription string, policyID *string) resource.TestCheckFunc {

	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)
		networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("(testAccCheckFirewallExists) Error creating OpenStack networking client: %s", err)
		}

		var found *firewalls.Firewall
		for i := 0; i < 5; i++ {
			// Firewall creation is asynchronous. Retry some times
			// if we get a 404 error. Fail on any other error.
			found, err = firewalls.Get(networkingClient, rs.Primary.ID).Extract()
			if err != nil {
				httpError, ok := err.(*gophercloud.UnexpectedResponseCodeError)
				if !ok || httpError.Actual != 404 {
					time.Sleep(time.Second)
					continue
				}
			}
			break
		}

		if err != nil {
			return err
		}

		if found.Name != expectedName {
			return fmt.Errorf("Expected Name to be <%s> but found <%s>", expectedName, found.Name)
		}
		if found.Description != expectedDescription {
			return fmt.Errorf("Expected Description to be <%s> but found <%s>", expectedDescription, found.Description)
		}
		if found.PolicyID == "" {
			return fmt.Errorf("Policy should not be empty")
		}
		if policyID != nil && found.PolicyID == *policyID {
			return fmt.Errorf("Policy had not been correctly updated. Went from <%s> to <%s>", expectedName, found.Name)
		}

		policyID = &found.PolicyID

		return nil
	}
}

const testFirewallConfig = `
resource "openstack_fw_firewall_v1" "accept_test" {
	policy_id = "${openstack_fw_policy_v1.accept_test_policy_1.id}"
}

resource "openstack_fw_policy_v1" "accept_test_policy_1" {
	name = "policy-1"
}
`

const testFirewallConfigUpdated = `
resource "openstack_fw_firewall_v1" "accept_test" {
	name = "accept_test"
	description = "terraform acceptance test"
	policy_id = "${openstack_fw_policy_v1.accept_test_policy_2.id}"
}

resource "openstack_fw_policy_v1" "accept_test_policy_2" {
	name = "policy-2"
}
`
