package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccOPCSecRule_Basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOPCSecRuleBasic, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSecRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccCheckSecRuleExists,
			},
		},
	})
}

func TestAccOPCSecRule_Complete(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOPCSecRuleComplete, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSecRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccCheckSecRuleExists,
			},
		},
	})
}

func testAccCheckSecRuleExists(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).SecRules()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_sec_rule" {
			continue
		}

		input := compute.GetSecRuleInput{
			Name: rs.Primary.Attributes["name"],
		}
		if _, err := client.GetSecRule(&input); err != nil {
			return fmt.Errorf("Error retrieving state of Sec Rule %s: %s", input.Name, err)
		}
	}

	return nil
}

func testAccCheckSecRuleDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).SecRules()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_sec_rule" {
			continue
		}

		input := compute.GetSecRuleInput{
			Name: rs.Primary.Attributes["name"],
		}
		if info, err := client.GetSecRule(&input); err == nil {
			return fmt.Errorf("Sec Rule %s still exists: %#v", input.Name, info)
		}
	}

	return nil
}

var testAccOPCSecRuleBasic = `
resource "opc_compute_security_list" "test" {
	name                 = "acc-test-sec-rule-list-%d"
        policy               = "PERMIT"
        outbound_cidr_policy = "DENY"
}

resource "opc_compute_security_application" "test" {
	name     = "acc-test-sec-rule-app-%d"
	protocol = "tcp"
	dport    = "8080"
}

resource "opc_compute_security_ip_list" "test" {
	name       = "acc-test-sec-rule-ip-list-%d"
	ip_entries = ["217.138.34.4"]
}

resource "opc_compute_sec_rule" "test" {
	name             = "acc-test-sec-rule-%d"
	source_list      = "seclist:${opc_compute_security_list.test.name}"
	destination_list = "seciplist:${opc_compute_security_ip_list.test.name}"
	action           = "PERMIT"
	application      = "${opc_compute_security_application.test.name}"
}
`

var testAccOPCSecRuleComplete = `
resource "opc_compute_security_list" "test" {
	name                 = "acc-test-sec-rule-list-%d"
        policy               = "PERMIT"
        outbound_cidr_policy = "DENY"
}

resource "opc_compute_security_application" "test" {
	name     = "acc-test-sec-rule-app-%d"
	protocol = "tcp"
	dport    = "8080"
}

resource "opc_compute_security_ip_list" "test" {
	name       = "acc-test-sec-rule-ip-list-%d"
	ip_entries = ["217.138.34.4"]
}

resource "opc_compute_sec_rule" "test" {
	name             = "acc-test-sec-rule-%d"
	source_list      = "seclist:${opc_compute_security_list.test.name}"
	destination_list = "seciplist:${opc_compute_security_ip_list.test.name}"
	action           = "PERMIT"
	application      = "${opc_compute_security_application.test.name}"
	disabled         = false
	description      = "This is a test description"
}
`
