package cloudstack

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func TestAccCloudStackNetworkACLRule_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackNetworkACLRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackNetworkACLRule_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackNetworkACLRulesExist("cloudstack_network_acl.foo"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.#", "3"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.3247834462.action", "allow"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.3247834462.source_cidr", "172.16.100.0/24"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.3247834462.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.3247834462.ports.#", "2"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.3247834462.ports.1889509032", "80"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.3247834462.ports.3638101695", "443"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.3247834462.traffic_type", "ingress"),
				),
			},
		},
	})
}

func TestAccCloudStackNetworkACLRule_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackNetworkACLRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackNetworkACLRule_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackNetworkACLRulesExist("cloudstack_network_acl.foo"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.#", "3"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.3247834462.action", "allow"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.3247834462.source_cidr", "172.16.100.0/24"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.3247834462.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.3247834462.ports.#", "2"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.3247834462.ports.1889509032", "80"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.3247834462.ports.3638101695", "443"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.3247834462.traffic_type", "ingress"),
				),
			},

			resource.TestStep{
				Config: testAccCloudStackNetworkACLRule_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackNetworkACLRulesExist("cloudstack_network_acl.foo"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.#", "4"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.3247834462.action", "allow"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.3247834462.source_cidr", "172.16.100.0/24"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.3247834462.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.3247834462.ports.#", "2"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.3247834462.ports.1889509032", "80"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.3247834462.ports.3638101695", "443"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.3247834462.traffic_type", "ingress"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.4267872693.action", "deny"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.4267872693.source_cidr", "10.0.0.0/24"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.4267872693.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.4267872693.ports.#", "2"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.4267872693.ports.1209010669", "1000-2000"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.4267872693.ports.1889509032", "80"),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl_rule.foo", "rule.4267872693.traffic_type", "egress"),
				),
			},
		},
	})
}

func testAccCheckCloudStackNetworkACLRulesExist(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No network ACL rule ID is set")
		}

		for k, uuid := range rs.Primary.Attributes {
			if !strings.Contains(k, ".uuids.") || strings.HasSuffix(k, ".uuids.#") {
				continue
			}

			cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)
			_, count, err := cs.NetworkACL.GetNetworkACLByID(uuid)

			if err != nil {
				return err
			}

			if count == 0 {
				return fmt.Errorf("Network ACL rule %s not found", k)
			}
		}

		return nil
	}
}

func testAccCheckCloudStackNetworkACLRuleDestroy(s *terraform.State) error {
	cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudstack_network_acl_rule" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No network ACL rule ID is set")
		}

		for k, uuid := range rs.Primary.Attributes {
			if !strings.Contains(k, ".uuids.") || strings.HasSuffix(k, ".uuids.#") {
				continue
			}

			_, _, err := cs.NetworkACL.GetNetworkACLByID(uuid)
			if err == nil {
				return fmt.Errorf("Network ACL rule %s still exists", rs.Primary.ID)
			}
		}
	}

	return nil
}

var testAccCloudStackNetworkACLRule_basic = fmt.Sprintf(`
resource "cloudstack_vpc" "foobar" {
  name = "terraform-vpc"
  cidr = "%s"
  vpc_offering = "%s"
  zone = "%s"
}

resource "cloudstack_network_acl" "foo" {
  name = "terraform-acl"
  description = "terraform-acl-text"
  vpc = "${cloudstack_vpc.foobar.id}"
}

resource "cloudstack_network_acl_rule" "foo" {
  aclid = "${cloudstack_network_acl.foo.id}"

  rule {
  	action = "allow"
    source_cidr = "172.18.100.0/24"
    protocol = "all"
    traffic_type = "ingress"
  }

  rule {
  	action = "allow"
    source_cidr = "172.18.100.0/24"
    protocol = "icmp"
    icmp_type = "-1"
    icmp_code = "-1"
    traffic_type = "ingress"
  }

  rule {
    source_cidr = "172.16.100.0/24"
    protocol = "tcp"
    ports = ["80", "443"]
    traffic_type = "ingress"
  }
}`,
	CLOUDSTACK_VPC_CIDR_1,
	CLOUDSTACK_VPC_OFFERING,
	CLOUDSTACK_ZONE)

var testAccCloudStackNetworkACLRule_update = fmt.Sprintf(`
resource "cloudstack_vpc" "foobar" {
  name = "terraform-vpc"
  cidr = "%s"
  vpc_offering = "%s"
  zone = "%s"
}

resource "cloudstack_network_acl" "foo" {
  name = "terraform-acl"
  description = "terraform-acl-text"
  vpc = "${cloudstack_vpc.foobar.id}"
}

resource "cloudstack_network_acl_rule" "foo" {
  aclid = "${cloudstack_network_acl.foo.id}"

  rule {
  	action = "deny"
    source_cidr = "172.18.100.0/24"
    protocol = "all"
    traffic_type = "ingress"
  }

  rule {
  	action = "deny"
    source_cidr = "172.18.100.0/24"
    protocol = "icmp"
    icmp_type = "-1"
    icmp_code = "-1"
    traffic_type = "ingress"
  }

  rule {
	  action = "allow"
    source_cidr = "172.16.100.0/24"
    protocol = "tcp"
    ports = ["80", "443"]
    traffic_type = "ingress"
  }

  rule {
	  action = "deny"
    source_cidr = "10.0.0.0/24"
    protocol = "tcp"
    ports = ["80", "1000-2000"]
    traffic_type = "egress"
  }
}`,
	CLOUDSTACK_VPC_CIDR_1,
	CLOUDSTACK_VPC_OFFERING,
	CLOUDSTACK_ZONE)
