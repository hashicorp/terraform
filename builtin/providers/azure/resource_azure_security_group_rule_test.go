package azure

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureSecurityGroupRule(t *testing.T) {
	name := "azure_security_group_rule.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureSecurityGroupRuleDeleted,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureSecurityGroupRule,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureSecurityGroupRuleExists(name),
					resource.TestCheckResourceAttr(name, "name", "terraform-secgroup-rule"),
					resource.TestCheckResourceAttr(name, "security_group_name", testAccSecurityGroupName),
					resource.TestCheckResourceAttr(name, "type", "Inbound"),
					resource.TestCheckResourceAttr(name, "action", "Deny"),
					resource.TestCheckResourceAttr(name, "priority", "200"),
					resource.TestCheckResourceAttr(name, "source_address_prefix", "100.0.0.0/32"),
					resource.TestCheckResourceAttr(name, "source_port_range", "1000"),
					resource.TestCheckResourceAttr(name, "destination_address_prefix", "10.0.0.0/32"),
					resource.TestCheckResourceAttr(name, "protocol", "TCP"),
				),
			},
		},
	})
}

func testAccCheckAzureSecurityGroupRuleExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Azure security group rule not found: %s", name)
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("Azure network security group rule ID not set: %s", name)
		}

		secGroupClient := testAccProvider.Meta().(*Client).secGroupClient

		secGroup, err := secGroupClient.GetNetworkSecurityGroup(testAccSecurityGroupName)
		if err != nil {
			return fmt.Errorf("Failed getting network security group details: %s", err)
		}

		for _, rule := range secGroup.Rules {
			if rule.Name == resource.Primary.ID {
				return nil
			}
		}

		return fmt.Errorf("Azure security group rule doesn't exist: %s", name)
	}
}

func testAccCheckAzureSecurityGroupRuleDeleted(s *terraform.State) error {
	for _, resource := range s.RootModule().Resources {
		if resource.Type != "azure_security_group_rule" {
			continue
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("Azure network security group ID not set.")
		}

		secGroupClient := testAccProvider.Meta().(*Client).secGroupClient

		secGroup, err := secGroupClient.GetNetworkSecurityGroup(testAccSecurityGroupName)
		if err != nil {
			return fmt.Errorf("Failed getting network security group details: %s", err)
		}

		for _, rule := range secGroup.Rules {
			if rule.Name == resource.Primary.ID {
				return fmt.Errorf("Azure network security group rule still exists!")
			}
		}
	}

	return nil
}

var testAccAzureSecurityGroupRule = testAccAzureSecurityGroupConfig + `
resource "azure_security_group_rule" "foo" {
	name = "terraform-secgroup-rule"
	security_group_name = "${azure_security_group.foo.name}"
	type = "Inbound"
	action = "Deny"
	priority = 200
	source_address_prefix = "100.0.0.0/32"
	source_port_range = "1000"
	destination_address_prefix = "10.0.0.0/32"
	destination_port_range = "1000"
	protocol = "TCP"
}
`
