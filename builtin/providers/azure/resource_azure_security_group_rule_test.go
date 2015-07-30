package azure

import (
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/management"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var (
	testAcctestingSecurityGroup1     = fmt.Sprintf("%s-%d", testAccSecurityGroupName, 1)
	testAccTestingSecurityGroupHash1 = fmt.Sprintf("%d", schema.HashString(testAcctestingSecurityGroup1))

	testAcctestingSecurityGroup2     = fmt.Sprintf("%s-%d", testAccSecurityGroupName, 2)
	testAccTestingSecurityGroupHash2 = fmt.Sprintf("%d", schema.HashString(testAcctestingSecurityGroup2))
)

func TestAccAzureSecurityGroupRuleBasic(t *testing.T) {
	name := "azure_security_group_rule.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureSecurityGroupRuleDeleted([]string{testAccSecurityGroupName}),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureSecurityGroupRuleBasicConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureSecurityGroupRuleExists(name, testAccSecurityGroupName),
					resource.TestCheckResourceAttr(name, "name", "terraform-secgroup-rule"),
					resource.TestCheckResourceAttr(name,
						fmt.Sprintf("security_group_names.%d", schema.HashString(testAccSecurityGroupName)),
						testAccSecurityGroupName),
					resource.TestCheckResourceAttr(name, "type", "Inbound"),
					resource.TestCheckResourceAttr(name, "action", "Deny"),
					resource.TestCheckResourceAttr(name, "priority", "200"),
					resource.TestCheckResourceAttr(name, "source_address_prefix", "100.0.0.0/32"),
					resource.TestCheckResourceAttr(name, "source_port_range", "1000"),
					resource.TestCheckResourceAttr(name, "destination_address_prefix", "10.0.0.0/32"),
					resource.TestCheckResourceAttr(name, "destination_port_range", "1000"),
					resource.TestCheckResourceAttr(name, "protocol", "TCP"),
				),
			},
		},
	})
}

func TestAccAzureSecurityGroupRuleAdvanced(t *testing.T) {
	name := "azure_security_group_rule.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: testAccCheckAzureSecurityGroupRuleDeleted(
			[]string{
				testAcctestingSecurityGroup1,
				testAcctestingSecurityGroup2,
			},
		),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureSecurityGroupRuleAdvancedConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureSecurityGroupRuleExists(name, testAcctestingSecurityGroup1),
					testAccCheckAzureSecurityGroupRuleExists(name, testAcctestingSecurityGroup2),
					resource.TestCheckResourceAttr(name, "name", "terraform-secgroup-rule"),
					resource.TestCheckResourceAttr(name, fmt.Sprintf("security_group_names.%s",
						testAccTestingSecurityGroupHash1), testAcctestingSecurityGroup1),
					resource.TestCheckResourceAttr(name, fmt.Sprintf("security_group_names.%s",
						testAccTestingSecurityGroupHash2), testAcctestingSecurityGroup2),
					resource.TestCheckResourceAttr(name, "type", "Inbound"),
					resource.TestCheckResourceAttr(name, "action", "Deny"),
					resource.TestCheckResourceAttr(name, "priority", "200"),
					resource.TestCheckResourceAttr(name, "source_address_prefix", "100.0.0.0/32"),
					resource.TestCheckResourceAttr(name, "source_port_range", "1000"),
					resource.TestCheckResourceAttr(name, "destination_address_prefix", "10.0.0.0/32"),
					resource.TestCheckResourceAttr(name, "destination_port_range", "1000"),
					resource.TestCheckResourceAttr(name, "protocol", "TCP"),
				),
			},
		},
	})
}

func TestAccAzureSecurityGroupRuleUpdate(t *testing.T) {
	name := "azure_security_group_rule.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: testAccCheckAzureSecurityGroupRuleDeleted(
			[]string{
				testAcctestingSecurityGroup1,
				testAcctestingSecurityGroup2,
			},
		),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureSecurityGroupRuleAdvancedConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureSecurityGroupRuleExists(name, testAcctestingSecurityGroup1),
					testAccCheckAzureSecurityGroupRuleExists(name, testAcctestingSecurityGroup2),
					resource.TestCheckResourceAttr(name, "name", "terraform-secgroup-rule"),
					resource.TestCheckResourceAttr(name, fmt.Sprintf("security_group_names.%s",
						testAccTestingSecurityGroupHash1), testAcctestingSecurityGroup1),
					resource.TestCheckResourceAttr(name, fmt.Sprintf("security_group_names.%s",
						testAccTestingSecurityGroupHash2), testAcctestingSecurityGroup2),
					resource.TestCheckResourceAttr(name, "type", "Inbound"),
					resource.TestCheckResourceAttr(name, "action", "Deny"),
					resource.TestCheckResourceAttr(name, "priority", "200"),
					resource.TestCheckResourceAttr(name, "source_address_prefix", "100.0.0.0/32"),
					resource.TestCheckResourceAttr(name, "source_port_range", "1000"),
					resource.TestCheckResourceAttr(name, "destination_address_prefix", "10.0.0.0/32"),
					resource.TestCheckResourceAttr(name, "destination_port_range", "1000"),
					resource.TestCheckResourceAttr(name, "protocol", "TCP"),
				),
			},

			resource.TestStep{
				Config: testAccAzureSecurityGroupRuleUpdateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureSecurityGroupRuleExists(name, testAcctestingSecurityGroup2),
					resource.TestCheckResourceAttr(name, "name", "terraform-secgroup-rule"),
					resource.TestCheckResourceAttr(name, fmt.Sprintf("security_group_names.%s",
						testAccTestingSecurityGroupHash2), testAcctestingSecurityGroup2),
					resource.TestCheckResourceAttr(name, "type", "Outbound"),
					resource.TestCheckResourceAttr(name, "action", "Allow"),
					resource.TestCheckResourceAttr(name, "priority", "100"),
					resource.TestCheckResourceAttr(name, "source_address_prefix", "101.0.0.0/32"),
					resource.TestCheckResourceAttr(name, "source_port_range", "1000"),
					resource.TestCheckResourceAttr(name, "destination_address_prefix", "10.0.0.0/32"),
					resource.TestCheckResourceAttr(name, "destination_port_range", "1001"),
					resource.TestCheckResourceAttr(name, "protocol", "UDP"),
				),
			},
		},
	})
}

func testAccCheckAzureSecurityGroupRuleExists(name, groupName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Azure security group rule not found: %s", name)
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("Azure network security group rule ID not set: %s", name)
		}

		secGroupClient := testAccProvider.Meta().(*Client).secGroupClient

		secGroup, err := secGroupClient.GetNetworkSecurityGroup(groupName)
		if err != nil {
			return fmt.Errorf("Failed getting network security group details for %q: %s", groupName, err)
		}

		for _, rule := range secGroup.Rules {
			if rule.Name == resource.Primary.ID {
				return nil
			}
		}

		return fmt.Errorf("Azure security group rule doesn't exist: %s", name)
	}
}

func testAccCheckAzureSecurityGroupRuleDeleted(groups []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, resource := range s.RootModule().Resources {
			if resource.Type != "azure_security_group_rule" {
				continue
			}

			if resource.Primary.ID == "" {
				return fmt.Errorf("Azure network security group ID not set.")
			}

			secGroupClient := testAccProvider.Meta().(*Client).secGroupClient

			for _, groupName := range groups {
				secGroup, err := secGroupClient.GetNetworkSecurityGroup(groupName)
				if err != nil {
					if !management.IsResourceNotFoundError(err) {
						return fmt.Errorf("Failed getting network security group details for %q: %s", groupName, err)
					}
				}

				for _, rule := range secGroup.Rules {
					if rule.Name == resource.Primary.ID {
						return fmt.Errorf("Azure network security group rule still exists!")
					}
				}
			}
		}

		return nil
	}
}

var testAccAzureSecurityGroupRuleBasicConfig = testAccAzureSecurityGroupConfig + `
resource "azure_security_group_rule" "foo" {
	name = "terraform-secgroup-rule"
	security_group_names = ["${azure_security_group.foo.name}"]
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
var testAccAzureSecurityGroupRuleAdvancedConfig = fmt.Sprintf(testAccAzureSecurityGroupConfigTemplate, "foo", testAcctestingSecurityGroup1) +
	fmt.Sprintf(testAccAzureSecurityGroupConfigTemplate, "bar", testAcctestingSecurityGroup2) + `
resource "azure_security_group_rule" "foo" {
	name = "terraform-secgroup-rule"
	security_group_names = ["${azure_security_group.foo.name}", "${azure_security_group.bar.name}"]
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

var testAccAzureSecurityGroupRuleUpdateConfig = fmt.Sprintf(testAccAzureSecurityGroupConfigTemplate, "foo", testAcctestingSecurityGroup1) +
	fmt.Sprintf(testAccAzureSecurityGroupConfigTemplate, "bar", testAcctestingSecurityGroup2) + `
resource "azure_security_group_rule" "foo" {
	name = "terraform-secgroup-rule"
	security_group_names = ["${azure_security_group.bar.name}"]
	type = "Outbound"
	action = "Allow"
	priority = 100
	source_address_prefix = "101.0.0.0/32"
	source_port_range = "1000"
	destination_address_prefix = "10.0.0.0/32"
	destination_port_range = "1001"
	protocol = "UDP"
}
`
