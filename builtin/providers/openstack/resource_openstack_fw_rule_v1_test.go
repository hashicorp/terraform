package openstack

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/fwaas/rules"
)

func TestAccFWRuleV1_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckFWRuleV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testFirewallRuleMinimalConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFWRuleV1Exists(
						"openstack_fw_rule_v1.accept_test_minimal",
						&rules.Rule{
							Protocol:  "udp",
							Action:    "deny",
							IPVersion: 4,
							Enabled:   true,
						}),
				),
			},
			resource.TestStep{
				Config: testFirewallRuleConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFWRuleV1Exists(
						"openstack_fw_rule_v1.accept_test",
						&rules.Rule{
							Name:                 "accept_test",
							Protocol:             "udp",
							Action:               "deny",
							Description:          "Terraform accept test",
							IPVersion:            4,
							SourceIPAddress:      "1.2.3.4",
							DestinationIPAddress: "4.3.2.0/24",
							SourcePort:           "444",
							DestinationPort:      "555",
							Enabled:              true,
						}),
				),
			},
			resource.TestStep{
				Config: testFirewallRuleUpdateAllFieldsConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFWRuleV1Exists(
						"openstack_fw_rule_v1.accept_test",
						&rules.Rule{
							Name:                 "accept_test_updated_2",
							Protocol:             "tcp",
							Action:               "allow",
							Description:          "Terraform accept test updated",
							IPVersion:            4,
							SourceIPAddress:      "1.2.3.0/24",
							DestinationIPAddress: "4.3.2.8",
							SourcePort:           "666",
							DestinationPort:      "777",
							Enabled:              false,
						}),
				),
			},
		},
	})
}

func testAccCheckFWRuleV1Destroy(s *terraform.State) error {

	config := testAccProvider.Meta().(*Config)
	networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckOpenstackFirewallRuleDestroy) Error creating OpenStack networking client: %s", err)
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_firewall_rule" {
			continue
		}
		_, err = rules.Get(networkingClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Firewall rule (%s) still exists.", rs.Primary.ID)
		}
		httpError, ok := err.(*gophercloud.UnexpectedResponseCodeError)
		if !ok || httpError.Actual != 404 {
			return httpError
		}
	}
	return nil
}

func testAccCheckFWRuleV1Exists(n string, expected *rules.Rule) resource.TestCheckFunc {

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
			return fmt.Errorf("(testAccCheckFirewallRuleExists) Error creating OpenStack networking client: %s", err)
		}

		var found *rules.Rule
		for i := 0; i < 5; i++ {
			// Firewall rule creation is asynchronous. Retry some times
			// if we get a 404 error. Fail on any other error.
			found, err = rules.Get(networkingClient, rs.Primary.ID).Extract()
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

		expected.ID = found.ID
		// Erase the tenant id because we don't want to compare
		// it as long it is not present in the expected
		found.TenantID = ""

		if !reflect.DeepEqual(expected, found) {
			return fmt.Errorf("Expected:\n%#v\nFound:\n%#v", expected, found)
		}

		return nil
	}
}

const testFirewallRuleMinimalConfig = `
resource "openstack_fw_rule_v1" "accept_test_minimal" {
	protocol = "udp"
	action = "deny"
}
`

const testFirewallRuleConfig = `
resource "openstack_fw_rule_v1" "accept_test" {
	name = "accept_test"
	description = "Terraform accept test"
	protocol = "udp"
	action = "deny"
	ip_version = 4
	source_ip_address = "1.2.3.4"
	destination_ip_address = "4.3.2.0/24"
	source_port = "444"
	destination_port = "555"
	enabled = true
}
`

const testFirewallRuleUpdateAllFieldsConfig = `
resource "openstack_fw_rule_v1" "accept_test" {
	name = "accept_test_updated_2"
	description = "Terraform accept test updated"
	protocol = "tcp"
	action = "allow"
	ip_version = 4
	source_ip_address = "1.2.3.0/24"
	destination_ip_address = "4.3.2.8"
	source_port = "666"
	destination_port = "777"
	enabled = false
}
`
