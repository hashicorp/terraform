package triton

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/joyent/triton-go"
)

func TestAccTritonFirewallRule_basic(t *testing.T) {
	config := testAccTritonFirewallRule_basic

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonFirewallRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonFirewallRuleExists("triton_firewall_rule.test"),
				),
			},
		},
	})
}

func TestAccTritonFirewallRule_update(t *testing.T) {
	preConfig := testAccTritonFirewallRule_basic
	postConfig := testAccTritonFirewallRule_update

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonFirewallRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonFirewallRuleExists("triton_firewall_rule.test"),
					resource.TestCheckResourceAttr("triton_firewall_rule.test", "rule", "FROM any TO tag \"www\" ALLOW tcp PORT 80"),
					resource.TestCheckResourceAttr("triton_firewall_rule.test", "enabled", "false"),
				),
			},

			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonFirewallRuleExists("triton_firewall_rule.test"),
					resource.TestCheckResourceAttr("triton_firewall_rule.test", "rule", "FROM any TO tag \"www\" BLOCK tcp PORT 80"),
					resource.TestCheckResourceAttr("triton_firewall_rule.test", "enabled", "true"),
				),
			},
		},
	})
}

func TestAccTritonFirewallRule_enable(t *testing.T) {
	preConfig := testAccTritonFirewallRule_basic
	postConfig := testAccTritonFirewallRule_enable

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonFirewallRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonFirewallRuleExists("triton_firewall_rule.test"),
					resource.TestCheckResourceAttr("triton_firewall_rule.test", "rule", "FROM any TO tag \"www\" ALLOW tcp PORT 80"),
					resource.TestCheckResourceAttr("triton_firewall_rule.test", "enabled", "false"),
				),
			},

			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonFirewallRuleExists("triton_firewall_rule.test"),
					resource.TestCheckResourceAttr("triton_firewall_rule.test", "rule", "FROM any TO tag \"www\" ALLOW tcp PORT 80"),
					resource.TestCheckResourceAttr("triton_firewall_rule.test", "enabled", "true"),
				),
			},
		},
	})
}

func testCheckTritonFirewallRuleExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		conn := testAccProvider.Meta().(*triton.Client)

		resp, err := conn.Firewall().GetFirewallRule(context.Background(), &triton.GetFirewallRuleInput{
			ID: rs.Primary.ID,
		})
		if err != nil && triton.IsResourceNotFound(err) {
			return fmt.Errorf("Bad: Check Firewall Rule Exists: %s", err)
		} else if err != nil {
			return err
		}

		if resp == nil {
			return fmt.Errorf("Bad: Firewall Rule %q does not exist", rs.Primary.ID)
		}

		return nil
	}
}

func testCheckTritonFirewallRuleDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*triton.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "triton_firewall_rule" {
			continue
		}

		resp, err := conn.Firewall().GetFirewallRule(context.Background(), &triton.GetFirewallRuleInput{
			ID: rs.Primary.ID,
		})
		if triton.IsResourceNotFound(err) {
			return nil
		} else if err != nil {
			return err
		}

		if resp != nil {
			return fmt.Errorf("Bad: Firewall Rule %q still exists", rs.Primary.ID)
		}
	}

	return nil
}

var testAccTritonFirewallRule_basic = `
resource "triton_firewall_rule" "test" {
	rule = "FROM any TO tag \"www\" ALLOW tcp PORT 80"
	enabled = false
}
`

var testAccTritonFirewallRule_update = `
resource "triton_firewall_rule" "test" {
	rule = "FROM any TO tag \"www\" BLOCK tcp PORT 80"
	enabled = true
}
`

var testAccTritonFirewallRule_enable = `
resource "triton_firewall_rule" "test" {
	rule = "FROM any TO tag \"www\" ALLOW tcp PORT 80"
	enabled = true
}
`
