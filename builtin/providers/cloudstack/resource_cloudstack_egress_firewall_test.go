package cloudstack

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func TestAccCloudStackEgressFirewall_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackEgressFirewallDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackEgressFirewall_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackEgressFirewallRulesExist("cloudstack_egress_firewall.foo"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "network", CLOUDSTACK_NETWORK_1),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo",
						"rule.411689741.source_cidr",
						CLOUDSTACK_NETWORK_1_IPADDRESS+"/32"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule.411689741.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule.411689741.ports.#", "2"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule.411689741.ports.1209010669", "1000-2000"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule.411689741.ports.1889509032", "80"),
				),
			},
		},
	})
}

func TestAccCloudStackEgressFirewall_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackEgressFirewallDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackEgressFirewall_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackEgressFirewallRulesExist("cloudstack_egress_firewall.foo"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "network", CLOUDSTACK_NETWORK_1),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule.#", "1"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo",
						"rule.411689741.source_cidr",
						CLOUDSTACK_NETWORK_1_IPADDRESS+"/32"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule.411689741.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule.411689741.ports.#", "2"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule.411689741.ports.1209010669", "1000-2000"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule.411689741.ports.1889509032", "80"),
				),
			},

			resource.TestStep{
				Config: testAccCloudStackEgressFirewall_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackEgressFirewallRulesExist("cloudstack_egress_firewall.foo"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "network", CLOUDSTACK_NETWORK_1),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule.#", "2"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo",
						"rule.411689741.source_cidr",
						CLOUDSTACK_NETWORK_1_IPADDRESS+"/32"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule.411689741.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule.411689741.ports.#", "2"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule.411689741.ports.1209010669", "1000-2000"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule.411689741.ports.1889509032", "80"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo",
						"rule.845479598.source_cidr",
						CLOUDSTACK_NETWORK_1_IPADDRESS+"/32"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule.845479598.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule.845479598.ports.#", "1"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule.845479598.ports.3638101695", "443"),
				),
			},
		},
	})
}

func testAccCheckCloudStackEgressFirewallRulesExist(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No firewall ID is set")
		}

		for k, uuid := range rs.Primary.Attributes {
			if !strings.Contains(k, ".uuids.") || strings.HasSuffix(k, ".uuids.#") {
				continue
			}

			cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)
			_, count, err := cs.Firewall.GetEgressFirewallRuleByID(uuid)

			if err != nil {
				return err
			}

			if count == 0 {
				return fmt.Errorf("Firewall rule for %s not found", k)
			}
		}

		return nil
	}
}

func testAccCheckCloudStackEgressFirewallDestroy(s *terraform.State) error {
	cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudstack_egress_firewall" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No instance ID is set")
		}

		for k, uuid := range rs.Primary.Attributes {
			if !strings.Contains(k, ".uuids.") || strings.HasSuffix(k, ".uuids.#") {
				continue
			}

			_, _, err := cs.Firewall.GetEgressFirewallRuleByID(uuid)
			if err == nil {
				return fmt.Errorf("Egress rule %s still exists", rs.Primary.ID)
			}
		}
	}

	return nil
}

var testAccCloudStackEgressFirewall_basic = fmt.Sprintf(`
resource "cloudstack_egress_firewall" "foo" {
  network = "%s"

  rule {
    source_cidr = "%s/32"
    protocol = "tcp"
    ports = ["80", "1000-2000"]
  }
}`,
	CLOUDSTACK_NETWORK_1,
	CLOUDSTACK_NETWORK_1_IPADDRESS)

var testAccCloudStackEgressFirewall_update = fmt.Sprintf(`
resource "cloudstack_egress_firewall" "foo" {
  network = "%s"

  rule {
    source_cidr = "%s/32"
    protocol = "tcp"
    ports = ["80", "1000-2000"]
  }

  rule {
    source_cidr = "%s/32"
    protocol = "tcp"
    ports = ["443"]
  }
}`,
	CLOUDSTACK_NETWORK_1,
	CLOUDSTACK_NETWORK_1_IPADDRESS,
	CLOUDSTACK_NETWORK_1_IPADDRESS)
