package scaleway

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/scaleway/scaleway-cli/pkg/api"
)

func TestAccScalewaySecurityGroupRule_Basic(t *testing.T) {
	var group api.ScalewaySecurityGroups

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckScalewaySecurityGroupRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckScalewaySecurityGroupRuleConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewaySecurityGroupsExists("scaleway_security_group.base", &group),
					resource.TestCheckResourceAttr("scaleway_security_group_rule.http", "action", "accept"),
					resource.TestCheckResourceAttr("scaleway_security_group_rule.http", "direction", "inbound"),
					resource.TestCheckResourceAttr("scaleway_security_group_rule.http", "ip_range", "0.0.0.0/0"),
					resource.TestCheckResourceAttr("scaleway_security_group_rule.http", "protocol", "TCP"),
					resource.TestCheckResourceAttr("scaleway_security_group_rule.http", "port", "80"),
					resource.TestCheckResourceAttr("scaleway_security_group_rule.https", "action", "accept"),
					resource.TestCheckResourceAttr("scaleway_security_group_rule.https", "direction", "inbound"),
					resource.TestCheckResourceAttr("scaleway_security_group_rule.https", "ip_range", "0.0.0.0/0"),
					resource.TestCheckResourceAttr("scaleway_security_group_rule.https", "protocol", "TCP"),
					resource.TestCheckResourceAttr("scaleway_security_group_rule.https", "port", "443"),
					testAccCheckScalewaySecurityGroupRuleExists("scaleway_security_group_rule.http", &group),
					testAccCheckScalewaySecurityGroupRuleAttributes("scaleway_security_group_rule.http", &group),
				),
			},
		},
	})
}

func testAccCheckScalewaySecurityGroupsExists(n string, group *api.ScalewaySecurityGroups) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Security Group Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Security Group is set")
		}

		conn := testAccProvider.Meta().(*Client).scaleway
		resp, err := conn.GetASecurityGroup(rs.Primary.ID)

		if err != nil {
			return err
		}

		if resp.SecurityGroups.ID == rs.Primary.ID {
			*group = resp.SecurityGroups
			return nil
		}

		return fmt.Errorf("Security Group not found")
	}
}

func testAccCheckScalewaySecurityGroupRuleDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client).scaleway

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "scaleway" {
			continue
		}

		groups, err := client.GetSecurityGroups()
		if err != nil {
			return err
		}

		all_err := true
		for _, group := range groups.SecurityGroups {
			_, err := client.GetASecurityGroupRule(group.ID, rs.Primary.ID)
			all_err = all_err && err != nil
		}

		if !all_err {
			return fmt.Errorf("Security Group still exists")
		}
	}

	return nil
}

func testAccCheckScalewaySecurityGroupRuleAttributes(n string, group *api.ScalewaySecurityGroups) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Unknown resource: %s", n)
		}

		client := testAccProvider.Meta().(*Client).scaleway
		rule, err := client.GetASecurityGroupRule(group.ID, rs.Primary.ID)
		if err != nil {
			return err
		}

		if rule.Rules.Action != "accept" {
			return fmt.Errorf("Wrong rule action")
		}
		if rule.Rules.Direction != "inbound" {
			return fmt.Errorf("wrong rule direction")
		}
		if rule.Rules.IPRange != "0.0.0.0/0" {
			return fmt.Errorf("wrong rule IP Range")
		}
		if rule.Rules.Protocol != "TCP" {
			return fmt.Errorf("wrong rule protocol")
		}
		if rule.Rules.DestPortFrom != 80 {
			return fmt.Errorf("Wrong port")
		}

		return nil
	}
}

func testAccCheckScalewaySecurityGroupRuleExists(n string, group *api.ScalewaySecurityGroups) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Security Group Rule Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Security Group Rule ID is set")
		}

		client := testAccProvider.Meta().(*Client).scaleway
		rule, err := client.GetASecurityGroupRule(group.ID, rs.Primary.ID)

		if err != nil {
			return err
		}

		if rule.Rules.ID != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		return nil
	}
}

var testAccCheckScalewaySecurityGroupRuleConfig = `
resource "scaleway_security_group" "base" {
  name = "public"
  description = "public gateway"
}

resource "scaleway_security_group_rule" "http" {
  security_group = "${scaleway_security_group.base.id}"

  action = "accept"
  direction = "inbound"
  ip_range = "0.0.0.0/0"
  protocol = "TCP"
  port = 80
}

resource "scaleway_security_group_rule" "https" {
  security_group = "${scaleway_security_group.base.id}"

  action = "accept"
  direction = "inbound"
  ip_range = "0.0.0.0/0"
  protocol = "TCP"
  port = 443
}
`
