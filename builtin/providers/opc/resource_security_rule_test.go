package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccOPCSecurityRule_Basic(t *testing.T) {
	rInt := acctest.RandInt()
	resName := "opc_compute_security_rule.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSecurityRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOPCSecurityRuleConfig_Basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecurityRuleExists,
					resource.TestCheckResourceAttr(resName, "name", fmt.Sprintf("testing-security-rule-%d", rInt)),
				),
			},
			{
				Config: testAccOPCSecurityRuleConfig_BasicUpdate(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecurityRuleExists,
					resource.TestCheckResourceAttr(resName, "enabled", "false"),
				),
			},
		},
	})
}

func TestAccOPCSecurityRule_Full(t *testing.T) {
	rInt := acctest.RandInt()
	resName := "opc_compute_security_rule.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSecurityRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOPCSecurityRuleConfig_Full(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecurityRuleExists,
					resource.TestCheckResourceAttr(resName, "name", fmt.Sprintf("testing-security-rule-%d", rInt)),
					resource.TestCheckResourceAttr(resName, "acl", fmt.Sprintf("test-security-rule-%d", rInt)),
					resource.TestCheckResourceAttr(resName, "src_vnic_set", fmt.Sprintf("test-security-rule-src-%d", rInt)),
					resource.TestCheckResourceAttr(resName, "dst_vnic_set", fmt.Sprintf("test-security-rule-dst-%d", rInt)),
					resource.TestCheckResourceAttr(resName, "dst_ip_address_prefixes.0", fmt.Sprintf("test-security-rule-dst-%d", rInt)),
					resource.TestCheckResourceAttr(resName, "src_ip_address_prefixes.0", fmt.Sprintf("test-security-rule-src-%d", rInt)),
					resource.TestCheckResourceAttr(resName, "security_protocols.0", fmt.Sprintf("test-security-rule-%d", rInt)),
				),
			},
		},
	})
}

func testAccCheckSecurityRuleExists(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).SecurityRules()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_sec_rule" {
			continue
		}

		input := compute.GetSecurityRuleInput{
			Name: rs.Primary.Attributes["name"],
		}
		if _, err := client.GetSecurityRule(&input); err != nil {
			return fmt.Errorf("Error retrieving state of Security Rule %s: %s", input.Name, err)
		}
	}

	return nil
}

func testAccCheckSecurityRuleDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).SecurityRules()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_security_rule" {
			continue
		}

		input := compute.GetSecurityRuleInput{
			Name: rs.Primary.Attributes["name"],
		}
		if info, err := client.GetSecurityRule(&input); err == nil {
			return fmt.Errorf("Security Rule %s still exists: %#v", input.Name, info)
		}
	}

	return nil
}

func testAccOPCSecurityRuleConfig_Basic(rInt int) string {
	return fmt.Sprintf(`
resource "opc_compute_security_rule" "test" {
  name = "testing-security-rule-%d"
  description = "testing-desc-%d"
  flow_direction = "ingress"
}`, rInt, rInt)
}

func testAccOPCSecurityRuleConfig_BasicUpdate(rInt int) string {
	return fmt.Sprintf(`
resource "opc_compute_security_rule" "test" {
  name = "testing-security-rule-%d"
  description = "testing-desc-%d"
  flow_direction = "egress"
  enabled = false
}`, rInt, rInt)
}

func testAccOPCSecurityRuleConfig_Full(rInt int) string {
	return fmt.Sprintf(`
resource "opc_compute_acl" "test" {
  name = "test-security-rule-%d"
}

resource "opc_compute_vnic_set" "src" {
  name = "test-security-rule-src-%d"
}

resource "opc_compute_vnic_set" "dst" {
  name = "test-security-rule-dst-%d"
}

resource "opc_compute_security_protocol" "test" {
  name = "test-security-rule-%d"
}

resource "opc_compute_ip_address_prefix_set" "src" {
  name = "test-security-rule-src-%d"
}

resource "opc_compute_ip_address_prefix_set" "dst" {
  name = "test-security-rule-dst-%d"
}

resource "opc_compute_security_rule" "test" {
  name                    = "testing-security-rule-%d"
  description             = "testing-desc-%d"
  flow_direction          = "ingress"
  acl                     = "${opc_compute_acl.test.name}"
  src_vnic_set            = "${opc_compute_vnic_set.src.name}"
  dst_vnic_set            = "${opc_compute_vnic_set.dst.name}"
  dst_ip_address_prefixes = ["${opc_compute_ip_address_prefix_set.dst.name}"]
  src_ip_address_prefixes = ["${opc_compute_ip_address_prefix_set.src.name}"]
  security_protocols     =  ["${opc_compute_security_protocol.test.name}"]
}`, rInt, rInt, rInt, rInt, rInt, rInt, rInt, rInt)
}
