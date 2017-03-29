package opc

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"testing"
)

func TestAccOPCResourceSecurityRule_Basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: opcResourceCheck(
			ruleResourceName,
			testAccCheckRuleDestroyed),
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityRuleBasic,
				Check: resource.ComposeTestCheckFunc(
					opcResourceCheck(
						ruleResourceName,
						testAccCheckRuleExists),
				),
			},
		},
	})
}

func testAccCheckRuleExists(state *OPCResourceState) error {
	ruleName := getRuleName(state)

	if _, err := state.SecurityRules().GetSecurityRule(ruleName); err != nil {
		return fmt.Errorf("Error retrieving state of security rule %s: %s", ruleName, err)
	}

	return nil
}

func getRuleName(rs *OPCResourceState) string {
	return rs.Attributes["name"]
}

func testAccCheckRuleDestroyed(state *OPCResourceState) error {
	ruleName := getRuleName(state)
	if info, err := state.SecurityRules().GetSecurityRule(ruleName); err == nil {
		return fmt.Errorf("Rule %s still exists: %#v", ruleName, info)
	}

	return nil
}

const ruleName = "test_rule"
const secListName = "sec-list1"
const secIpListName = "sec-ip-list1"

var ruleResourceName = fmt.Sprintf("opc_compute_security_rule.%s", ruleName)

var testAccSecurityRuleBasic = fmt.Sprintf(`
resource "opc_compute_security_rule" "%s" {
	name = "test"
	source_list = "seclist:${opc_compute_security_list.sec-list1.name}"
	destination_list = "seciplist:${opc_compute_security_ip_list.sec-ip-list1.name}"
	action = "PERMIT"
	application = "${opc_compute_security_application.spring-boot.name}"
	disabled = false
}

resource "opc_compute_security_list" "%s" {
	name = "sec-list-1"
        policy = "PERMIT"
        outbound_cidr_policy = "DENY"
}

resource "opc_compute_security_application" "spring-boot" {
	name = "spring-boot"
	protocol = "tcp"
	dport = "8080"
}

resource "opc_compute_security_ip_list" "%s" {
	name = "sec-ip-list1"
	ip_entries = ["217.138.34.4"]
}
`, ruleName, secListName, secIpListName)
