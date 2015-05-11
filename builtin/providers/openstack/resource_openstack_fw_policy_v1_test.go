package openstack

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/fwaas/policies"
)

func TestAccFWPolicyV1_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckFWPolicyV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testFirewallPolicyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFWPolicyV1Exists(
						"openstack_fw_policy_v1.accept_test",
						"", "", 0),
				),
			},
		},
	})
}

func TestAccFWPolicyV1_addRules(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckFWPolicyV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testFirewallPolicyConfigAddRules,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFWPolicyV1Exists(
						"openstack_fw_policy_v1.accept_test",
						"accept_test", "terraform acceptance test", 2),
				),
			},
		},
	})
}

func TestAccFWPolicyV1_deleteRules(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckFWPolicyV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testFirewallPolicyUpdateDeleteRule,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFWPolicyV1Exists(
						"openstack_fw_policy_v1.accept_test",
						"accept_test", "terraform acceptance test", 1),
				),
			},
		},
	})
}

func testAccCheckFWPolicyV1Destroy(s *terraform.State) error {

	config := testAccProvider.Meta().(*Config)
	networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckOpenstackFirewallPolicyDestroy) Error creating OpenStack networking client: %s", err)
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_fw_policy_v1" {
			continue
		}
		_, err = policies.Get(networkingClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Firewall policy (%s) still exists.", rs.Primary.ID)
		}
		httpError, ok := err.(*gophercloud.UnexpectedResponseCodeError)
		if !ok || httpError.Actual != 404 {
			return httpError
		}
	}
	return nil
}

func testAccCheckFWPolicyV1Exists(n, name, description string, ruleCount int) resource.TestCheckFunc {

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

		if name != found.Name {
			return fmt.Errorf("Expected name <%s>, but found <%s>", name, found.Name)
		}

		if description != found.Description {
			return fmt.Errorf("Expected description <%s>, but found <%s>", description, found.Description)
		}

		if ruleCount != len(found.Rules) {
			return fmt.Errorf("Expected rule count <%d>, but found <%d>", ruleCount, len(found.Rules))
		}

		return nil
	}
}

const testFirewallPolicyConfig = `
resource "openstack_fw_policy_v1" "accept_test" {

}
`

const testFirewallPolicyConfigAddRules = `
resource "openstack_fw_policy_v1" "accept_test" {
	name = "accept_test"
	description =  "terraform acceptance test"
	rules = [
		"${openstack_fw_rule_v1.accept_test_udp_deny.id}",
		"${openstack_fw_rule_v1.accept_test_tcp_allow.id}"
	]
}

resource "openstack_fw_rule_v1" "accept_test_tcp_allow" {
	protocol = "tcp"
	action = "allow"
}

resource "openstack_fw_rule_v1" "accept_test_udp_deny" {
	protocol = "udp"
	action = "deny"
}
`

const testFirewallPolicyUpdateDeleteRule = `
resource "openstack_fw_policy_v1" "accept_test" {
	name = "accept_test"
	description =  "terraform acceptance test"
	rules = [
		"${openstack_fw_rule_v1.accept_test_udp_deny.id}"
	]
}

resource "openstack_fw_rule_v1" "accept_test_udp_deny" {
	protocol = "udp"
	action = "deny"
}
`
