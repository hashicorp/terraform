package openstack

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/fwaas/policies"
)

func TestAccOpenstackFirewallPolicy(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckOpenstackFirewallPolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testFirewallPolicyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFirewallPolicyExists(
						"openstack_fw_policy_v2.accept_test",
						&policies.Policy{
							Rules: []string{},
						}),
				),
			},
			resource.TestStep{
				Config: testFirewallPolicyConfigAddRules,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFirewallPolicyExists(
						"openstack_fw_policy_v2.accept_test",
						&policies.Policy{
							Name:        "accept_test",
							Description: "terraform acceptance test",
							Rules: []string{
								"",
								"",
							},
						}),
				),
			},
			resource.TestStep{
				Config: testFirewallPolicyUpdateDeleteRule,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFirewallPolicyExists(
						"openstack_fw_policy_v2.accept_test",
						&policies.Policy{}),
				),
			},
		},
	})
}

func testAccCheckOpenstackFirewallPolicyDestroy(s *terraform.State) error {

	config := testAccProvider.Meta().(*Config)
	networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckOpenstackFirewallPolicyDestroy) Error creating OpenStack networking client: %s", err)
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_fw_policy_v2" {
			continue
		}
		_, err = policies.Get(networkingClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Firewall policy (%s) still exists.", rs.Primary.ID)
		}
		httpError, ok := err.(*perigee.UnexpectedResponseCodeError)
		if !ok || httpError.Actual != 404 {
			return httpError
		}
	}
	return nil
}

func testAccCheckFirewallPolicyExists(n string, expected *policies.Policy) resource.TestCheckFunc {

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
			return fmt.Errorf("(testAccCheckFirewallPolicyExists) Error creating OpenStack networking client: %s", err)
		}

		var found *policies.Policy
		for i := 0; i < 5; i++ {
			// Firewall policy creation is asynchronous. Retry some times
			// if we get a 404 error. Fail on any other error.
			found, err = policies.Get(networkingClient, rs.Primary.ID).Extract()
			if err != nil {
				httpError, ok := err.(*perigee.UnexpectedResponseCodeError)
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

		expected.ID = found.ID

		if !reflect.DeepEqual(expected, found) {
			return fmt.Errorf("Expected:\n%#v\nFound:\n%#v", expected, found)
		}

		return nil
	}
}

const testFirewallPolicyConfig = `
resource "openstack_fw_policy_v2" "accept_test" {

}
`

const testFirewallPolicyConfigAddRules = `
resource "openstack_fw_policy_v2" "accept_test" {
	name = "accept_test"
	description =  "terraform acceptance test"
	rules = [
		"${openstack_fw_rule_v2.accept_test_udp_deny.id}",
		"${openstack_fw_rule_v2.accept_test_tcp_allow.id}",
		"${openstack_fw_rule_v2.accept_test_icmp_allow.id}"
	]
}

resource "openstack_fw_rule_v2" "accept_test_tcp_allow" {
	protocol = "tcp"
	action = "allow"
}

resource "openstack_fw_rule_v2" "accept_test_udp_deny" {
	protocol = "udp"
	action = "deny"
}

resource "openstack_fw_rule_v2" "accept_test_icmp_allow" {
	protocol = "icmp"
	action = "allow"
}
`

const testFirewallPolicyUpdateDeleteRule = `
resource "openstack_fw_policy_v2" "accept_test" {
	name = "accept_test"
	description =  "terraform acceptance test"
	rules = [
		"${openstack_fw_rule_v2.accept_test_udp_deny.id}"
	]
}

resource "openstack_fw_rule_v2" "accept_test_udp_deny" {
	protocol = "udp"
	action = "deny"
}
`
