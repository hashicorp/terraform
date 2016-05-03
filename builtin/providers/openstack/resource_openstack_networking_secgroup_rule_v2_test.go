package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/security/rules"
)

func TestAccNetworkingV2SecGroupRule_basic(t *testing.T) {
	var security_group_1 groups.SecGroup
	var security_group_2 groups.SecGroup
	var security_group_rule_1 rules.SecGroupRule
	var security_group_rule_2 rules.SecGroupRule

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2SecGroupRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2SecGroupRule_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2SecGroupExists(t, "openstack_networking_secgroup_v2.sg_foo", &security_group_1),
					testAccCheckNetworkingV2SecGroupExists(t, "openstack_networking_secgroup_v2.sg_bar", &security_group_2),
					testAccCheckNetworkingV2SecGroupRuleExists(t, "openstack_networking_secgroup_rule_v2.sr_foo", &security_group_rule_1),
					testAccCheckNetworkingV2SecGroupRuleExists(t, "openstack_networking_secgroup_rule_v2.sr_bar", &security_group_rule_2),
				),
			},
		},
	})
}

func testAccCheckNetworkingV2SecGroupRuleDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckNetworkingV2SecGroupRuleDestroy) Error creating OpenStack networking client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_networking_secgroup_rule_v2" {
			continue
		}

		_, err := rules.Get(networkingClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Security group rule still exists")
		}
	}

	return nil
}

func testAccCheckNetworkingV2SecGroupRuleExists(t *testing.T, n string, security_group_rule *rules.SecGroupRule) resource.TestCheckFunc {
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
			return fmt.Errorf("(testAccCheckNetworkingV2SecGroupRuleExists) Error creating OpenStack networking client: %s", err)
		}

		found, err := rules.Get(networkingClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Security group rule not found")
		}

		*security_group_rule = *found

		return nil
	}
}

var testAccNetworkingV2SecGroupRule_basic = fmt.Sprintf(`
  resource "openstack_networking_secgroup_v2" "sg_foo" {
    name = "security_group_1"
    description = "terraform security group rule acceptance test"
  }
  resource "openstack_networking_secgroup_v2" "sg_bar" {
    name = "security_group_2"
    description = "terraform security group rule acceptance test"
  }
  resource "openstack_networking_secgroup_rule_v2" "sr_foo" {
    direction = "ingress"
    ethertype = "IPv4"
    port_range_max = 22
    port_range_min = 22
    protocol = "tcp"
    remote_ip_prefix = "0.0.0.0/0"
    security_group_id = "${openstack_networking_secgroup_v2.sg_foo.id}"
  }
  resource "openstack_networking_secgroup_rule_v2" "sr_bar" {
    direction = "ingress"
    ethertype = "IPv4"
    port_range_max = 80 
    port_range_min = 80
    protocol = "tcp"
    remote_group_id = "${openstack_networking_secgroup_v2.sg_foo.id}"
    security_group_id = "${openstack_networking_secgroup_v2.sg_bar.id}"
  }`)
