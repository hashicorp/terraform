package cloudstack

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func TestAccCloudStackEgressFirewall_basic(t *testing.T) {
	hash := makeTestCloudStackEgressFirewallRuleHash([]interface{}{"1000-2000", "80"})

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
						"rule."+hash+".source_cidr",
						CLOUDSTACK_NETWORK_1_IPADDRESS+"/32"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule."+hash+".protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule."+hash+".ports.#", "2"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule."+hash+".ports.1209010669", "1000-2000"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule."+hash+".ports.1889509032", "80"),
				),
			},
		},
	})
}

func TestAccCloudStackEgressFirewall_update(t *testing.T) {
	hash1 := makeTestCloudStackEgressFirewallRuleHash([]interface{}{"1000-2000", "80"})
	hash2 := makeTestCloudStackEgressFirewallRuleHash([]interface{}{"443"})

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
						"rule."+hash1+".source_cidr",
						CLOUDSTACK_NETWORK_1_IPADDRESS+"/32"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule."+hash1+".protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule."+hash1+".ports.#", "2"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule."+hash1+".ports.1209010669", "1000-2000"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule."+hash1+".ports.1889509032", "80"),
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
						"rule."+hash1+".source_cidr",
						CLOUDSTACK_NETWORK_1_IPADDRESS+"/32"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule."+hash1+".protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule."+hash1+".ports.#", "2"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule."+hash1+".ports.1209010669", "1000-2000"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule."+hash1+".ports.1889509032", "80"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo",
						"rule."+hash2+".source_cidr",
						CLOUDSTACK_NETWORK_1_IPADDRESS+"/32"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule."+hash2+".protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule."+hash2+".ports.#", "1"),
					resource.TestCheckResourceAttr(
						"cloudstack_egress_firewall.foo", "rule."+hash2+".ports.3638101695", "443"),
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

func makeTestCloudStackEgressFirewallRuleHash(ports []interface{}) string {
	return strconv.Itoa(resourceCloudStackEgressFirewallRuleHash(map[string]interface{}{
		"source_cidr": CLOUDSTACK_NETWORK_1_IPADDRESS + "/32",
		"protocol":    "tcp",
		"ports":       schema.NewSet(schema.HashString, ports),
		"icmp_type":   0,
		"icmp_code":   0,
	}))
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
