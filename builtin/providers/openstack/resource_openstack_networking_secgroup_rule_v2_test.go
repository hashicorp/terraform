package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
)

func TestAccNetworkingV2SecGroupRule_basic(t *testing.T) {
	var secgroup_1 groups.SecGroup
	var secgroup_2 groups.SecGroup
	var secgroup_rule_1 rules.SecGroupRule
	var secgroup_rule_2 rules.SecGroupRule

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2SecGroupRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2SecGroupRule_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2SecGroupExists(
						"openstack_networking_secgroup_v2.secgroup_1", &secgroup_1),
					testAccCheckNetworkingV2SecGroupExists(
						"openstack_networking_secgroup_v2.secgroup_2", &secgroup_2),
					testAccCheckNetworkingV2SecGroupRuleExists(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_1", &secgroup_rule_1),
					testAccCheckNetworkingV2SecGroupRuleExists(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_2", &secgroup_rule_2),
				),
			},
		},
	})
}

func TestAccNetworkingV2SecGroupRule_lowerCaseCIDR(t *testing.T) {
	var secgroup_1 groups.SecGroup
	var secgroup_rule_1 rules.SecGroupRule

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2SecGroupRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2SecGroupRule_lowerCaseCIDR,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2SecGroupExists(
						"openstack_networking_secgroup_v2.secgroup_1", &secgroup_1),
					testAccCheckNetworkingV2SecGroupRuleExists(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_1", &secgroup_rule_1),
					resource.TestCheckResourceAttr(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_1", "remote_ip_prefix", "2001:558:fc00::/39"),
				),
			},
		},
	})
}

func TestAccNetworkingV2SecGroupRule_timeout(t *testing.T) {
	var secgroup_1 groups.SecGroup
	var secgroup_2 groups.SecGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2SecGroupRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2SecGroupRule_timeout,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2SecGroupExists(
						"openstack_networking_secgroup_v2.secgroup_1", &secgroup_1),
					testAccCheckNetworkingV2SecGroupExists(
						"openstack_networking_secgroup_v2.secgroup_2", &secgroup_2),
				),
			},
		},
	})
}

func TestAccNetworkingV2SecGroupRule_protocols(t *testing.T) {
	var secgroup_1 groups.SecGroup
	var secgroup_rule_ah rules.SecGroupRule
	var secgroup_rule_dccp rules.SecGroupRule
	var secgroup_rule_egp rules.SecGroupRule
	var secgroup_rule_esp rules.SecGroupRule
	var secgroup_rule_gre rules.SecGroupRule
	var secgroup_rule_igmp rules.SecGroupRule
	var secgroup_rule_ipv6_encap rules.SecGroupRule
	var secgroup_rule_ipv6_frag rules.SecGroupRule
	var secgroup_rule_ipv6_icmp rules.SecGroupRule
	var secgroup_rule_ipv6_nonxt rules.SecGroupRule
	var secgroup_rule_ipv6_opts rules.SecGroupRule
	var secgroup_rule_ipv6_route rules.SecGroupRule
	var secgroup_rule_ospf rules.SecGroupRule
	var secgroup_rule_pgm rules.SecGroupRule
	var secgroup_rule_rsvp rules.SecGroupRule
	var secgroup_rule_sctp rules.SecGroupRule
	var secgroup_rule_udplite rules.SecGroupRule
	var secgroup_rule_vrrp rules.SecGroupRule

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2SecGroupRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2SecGroupRule_protocols,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2SecGroupExists(
						"openstack_networking_secgroup_v2.secgroup_1", &secgroup_1),
					testAccCheckNetworkingV2SecGroupRuleExists(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_ah", &secgroup_rule_ah),
					testAccCheckNetworkingV2SecGroupRuleExists(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_dccp", &secgroup_rule_dccp),
					testAccCheckNetworkingV2SecGroupRuleExists(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_egp", &secgroup_rule_egp),
					testAccCheckNetworkingV2SecGroupRuleExists(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_esp", &secgroup_rule_esp),
					testAccCheckNetworkingV2SecGroupRuleExists(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_gre", &secgroup_rule_gre),
					testAccCheckNetworkingV2SecGroupRuleExists(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_igmp", &secgroup_rule_igmp),
					testAccCheckNetworkingV2SecGroupRuleExists(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_ipv6_encap", &secgroup_rule_ipv6_encap),
					testAccCheckNetworkingV2SecGroupRuleExists(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_ipv6_frag", &secgroup_rule_ipv6_frag),
					testAccCheckNetworkingV2SecGroupRuleExists(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_ipv6_icmp", &secgroup_rule_ipv6_icmp),
					testAccCheckNetworkingV2SecGroupRuleExists(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_ipv6_nonxt", &secgroup_rule_ipv6_nonxt),
					testAccCheckNetworkingV2SecGroupRuleExists(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_ipv6_opts", &secgroup_rule_ipv6_opts),
					testAccCheckNetworkingV2SecGroupRuleExists(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_ipv6_route", &secgroup_rule_ipv6_route),
					testAccCheckNetworkingV2SecGroupRuleExists(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_ospf", &secgroup_rule_ospf),
					testAccCheckNetworkingV2SecGroupRuleExists(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_pgm", &secgroup_rule_pgm),
					testAccCheckNetworkingV2SecGroupRuleExists(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_rsvp", &secgroup_rule_rsvp),
					testAccCheckNetworkingV2SecGroupRuleExists(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_sctp", &secgroup_rule_sctp),
					testAccCheckNetworkingV2SecGroupRuleExists(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_udplite", &secgroup_rule_udplite),
					testAccCheckNetworkingV2SecGroupRuleExists(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_vrrp", &secgroup_rule_vrrp),
					resource.TestCheckResourceAttr(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_ah", "protocol", "ah"),
					resource.TestCheckResourceAttr(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_dccp", "protocol", "dccp"),
					resource.TestCheckResourceAttr(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_egp", "protocol", "egp"),
					resource.TestCheckResourceAttr(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_esp", "protocol", "esp"),
					resource.TestCheckResourceAttr(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_gre", "protocol", "gre"),
					resource.TestCheckResourceAttr(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_igmp", "protocol", "igmp"),
					resource.TestCheckResourceAttr(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_ipv6_encap", "protocol", "ipv6-encap"),
					resource.TestCheckResourceAttr(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_ipv6_frag", "protocol", "ipv6-frag"),
					resource.TestCheckResourceAttr(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_ipv6_icmp", "protocol", "ipv6-icmp"),
					resource.TestCheckResourceAttr(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_ipv6_nonxt", "protocol", "ipv6-nonxt"),
					resource.TestCheckResourceAttr(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_ipv6_opts", "protocol", "ipv6-opts"),
					resource.TestCheckResourceAttr(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_ipv6_route", "protocol", "ipv6-route"),
					resource.TestCheckResourceAttr(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_ospf", "protocol", "ospf"),
					resource.TestCheckResourceAttr(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_pgm", "protocol", "pgm"),
					resource.TestCheckResourceAttr(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_rsvp", "protocol", "rsvp"),
					resource.TestCheckResourceAttr(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_sctp", "protocol", "sctp"),
					resource.TestCheckResourceAttr(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_udplite", "protocol", "udplite"),
					resource.TestCheckResourceAttr(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_vrrp", "protocol", "vrrp"),
				),
			},
		},
	})
}

func TestAccNetworkingV2SecGroupRule_numericProtocol(t *testing.T) {
	var secgroup_1 groups.SecGroup
	var secgroup_rule_1 rules.SecGroupRule

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2SecGroupRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2SecGroupRule_numericProtocol,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2SecGroupExists(
						"openstack_networking_secgroup_v2.secgroup_1", &secgroup_1),
					testAccCheckNetworkingV2SecGroupRuleExists(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_1", &secgroup_rule_1),
					resource.TestCheckResourceAttr(
						"openstack_networking_secgroup_rule_v2.secgroup_rule_1", "protocol", "115"),
				),
			},
		},
	})
}

func testAccCheckNetworkingV2SecGroupRuleDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
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

func testAccCheckNetworkingV2SecGroupRuleExists(n string, security_group_rule *rules.SecGroupRule) resource.TestCheckFunc {
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
			return fmt.Errorf("Error creating OpenStack networking client: %s", err)
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

const testAccNetworkingV2SecGroupRule_basic = `
resource "openstack_networking_secgroup_v2" "secgroup_1" {
  name = "secgroup_1"
  description = "terraform security group rule acceptance test"
}

resource "openstack_networking_secgroup_v2" "secgroup_2" {
  name = "secgroup_2"
  description = "terraform security group rule acceptance test"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_1" {
  direction = "ingress"
  ethertype = "IPv4"
  port_range_max = 22
  port_range_min = 22
  protocol = "tcp"
  remote_ip_prefix = "0.0.0.0/0"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_2" {
  direction = "ingress"
  ethertype = "IPv4"
  port_range_max = 80
  port_range_min = 80
  protocol = "tcp"
  remote_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_2.id}"
}
`

const testAccNetworkingV2SecGroupRule_lowerCaseCIDR = `
resource "openstack_networking_secgroup_v2" "secgroup_1" {
  name = "secgroup_1"
  description = "terraform security group rule acceptance test"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_1" {
  direction = "ingress"
  ethertype = "IPv6"
  port_range_max = 22
  port_range_min = 22
  protocol = "tcp"
  remote_ip_prefix = "2001:558:FC00::/39"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
}
`

const testAccNetworkingV2SecGroupRule_timeout = `
resource "openstack_networking_secgroup_v2" "secgroup_1" {
  name = "secgroup_1"
  description = "terraform security group rule acceptance test"
}

resource "openstack_networking_secgroup_v2" "secgroup_2" {
  name = "secgroup_2"
  description = "terraform security group rule acceptance test"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_1" {
  direction = "ingress"
  ethertype = "IPv4"
  port_range_max = 22
  port_range_min = 22
  protocol = "tcp"
  remote_ip_prefix = "0.0.0.0/0"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"

  timeouts {
    delete = "5m"
  }
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_2" {
  direction = "ingress"
  ethertype = "IPv4"
  port_range_max = 80
  port_range_min = 80
  protocol = "tcp"
  remote_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_2.id}"

  timeouts {
    delete = "5m"
  }
}
`

const testAccNetworkingV2SecGroupRule_protocols = `
resource "openstack_networking_secgroup_v2" "secgroup_1" {
  name = "secgroup_1"
  description = "terraform security group rule acceptance test"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_ah" {
  direction = "ingress"
  ethertype = "IPv4"
  protocol = "ah"
  remote_ip_prefix = "0.0.0.0/0"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_dccp" {
  direction = "ingress"
  ethertype = "IPv4"
  protocol = "dccp"
  remote_ip_prefix = "0.0.0.0/0"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_egp" {
  direction = "ingress"
  ethertype = "IPv4"
  protocol = "egp"
  remote_ip_prefix = "0.0.0.0/0"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_esp" {
  direction = "ingress"
  ethertype = "IPv4"
  protocol = "esp"
  remote_ip_prefix = "0.0.0.0/0"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_gre" {
  direction = "ingress"
  ethertype = "IPv4"
  protocol = "gre"
  remote_ip_prefix = "0.0.0.0/0"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_igmp" {
  direction = "ingress"
  ethertype = "IPv4"
  protocol = "igmp"
  remote_ip_prefix = "0.0.0.0/0"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_ipv6_encap" {
  direction = "ingress"
  ethertype = "IPv6"
  protocol = "ipv6-encap"
  remote_ip_prefix = "::/0"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_ipv6_frag" {
  direction = "ingress"
  ethertype = "IPv6"
  protocol = "ipv6-frag"
  remote_ip_prefix = "::/0"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_ipv6_icmp" {
  direction = "ingress"
  ethertype = "IPv6"
  protocol = "ipv6-icmp"
  remote_ip_prefix = "::/0"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_ipv6_nonxt" {
  direction = "ingress"
  ethertype = "IPv6"
  protocol = "ipv6-nonxt"
  remote_ip_prefix = "::/0"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_ipv6_opts" {
  direction = "ingress"
  ethertype = "IPv6"
  protocol = "ipv6-opts"
  remote_ip_prefix = "::/0"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_ipv6_route" {
  direction = "ingress"
  ethertype = "IPv6"
  protocol = "ipv6-route"
  remote_ip_prefix = "::/0"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_ospf" {
  direction = "ingress"
  ethertype = "IPv4"
  protocol = "ospf"
  remote_ip_prefix = "0.0.0.0/0"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_pgm" {
  direction = "ingress"
  ethertype = "IPv4"
  protocol = "pgm"
  remote_ip_prefix = "0.0.0.0/0"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_rsvp" {
  direction = "ingress"
  ethertype = "IPv4"
  protocol = "rsvp"
  remote_ip_prefix = "0.0.0.0/0"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_sctp" {
  direction = "ingress"
  ethertype = "IPv4"
  protocol = "sctp"
  remote_ip_prefix = "0.0.0.0/0"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_udplite" {
  direction = "ingress"
  ethertype = "IPv4"
  protocol = "udplite"
  remote_ip_prefix = "0.0.0.0/0"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_vrrp" {
  direction = "ingress"
  ethertype = "IPv4"
  protocol = "vrrp"
  remote_ip_prefix = "0.0.0.0/0"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
}
`

const testAccNetworkingV2SecGroupRule_numericProtocol = `
resource "openstack_networking_secgroup_v2" "secgroup_1" {
  name = "secgroup_1"
  description = "terraform security group rule acceptance test"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_1" {
  direction = "ingress"
  ethertype = "IPv4"
  port_range_max = 22
  port_range_min = 22
  protocol = "115"
  remote_ip_prefix = "0.0.0.0/0"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
}
`
