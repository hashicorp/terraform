package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMNetworkSecurityRule_basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMNetworkSecurityRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureRMNetworkSecurityRule_basic,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkSecurityRuleExists("azurerm_network_security_rule.test"),
				),
			},
		},
	})
}

func TestAccAzureRMNetworkSecurityRule_addingRules(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMNetworkSecurityRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureRMNetworkSecurityRule_updateBasic,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkSecurityRuleExists("azurerm_network_security_rule.test1"),
				),
			},

			resource.TestStep{
				Config: testAccAzureRMNetworkSecurityRule_updateExtraRule,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkSecurityRuleExists("azurerm_network_security_rule.test2"),
				),
			},
		},
	})
}

func testCheckAzureRMNetworkSecurityRuleExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		sgName := rs.Primary.Attributes["network_security_group_name"]
		sgrName := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for network security rule: %s", sgName)
		}

		conn := testAccProvider.Meta().(*ArmClient).secRuleClient

		resp, err := conn.Get(resourceGroup, sgName, sgrName)
		if err != nil {
			return fmt.Errorf("Bad: Get on secRuleClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Network Security Rule %q (resource group: %q) (network security group: %q) does not exist", sgrName, sgName, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMNetworkSecurityRuleDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).secRuleClient

	for _, rs := range s.RootModule().Resources {

		if rs.Type != "azurerm_network_security_rule" {
			continue
		}

		sgName := rs.Primary.Attributes["network_security_group_name"]
		sgrName := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, sgName, sgrName)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Network Security Rule still exists:\n%#v", resp.Properties)
		}
	}

	return nil
}

var testAccAzureRMNetworkSecurityRule_basic = `
resource "azurerm_resource_group" "test" {
    name = "acceptanceTestResourceGroup1"
    location = "West US"
}

resource "azurerm_network_security_group" "test" {
    name = "acceptanceTestSecurityGroup1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_network_security_rule" "test" {
	name = "test123"
    	priority = 100
    	direction = "Outbound"
    	access = "Allow"
    	protocol = "Tcp"
    	source_port_range = "*"
    	destination_port_range = "*"
    	source_address_prefix = "*"
    	destination_address_prefix = "*"
    	resource_group_name = "${azurerm_resource_group.test.name}"
    	network_security_group_name = "${azurerm_network_security_group.test.name}"
}
`

var testAccAzureRMNetworkSecurityRule_updateBasic = `
resource "azurerm_resource_group" "test1" {
    name = "acceptanceTestResourceGroup2"
    location = "West US"
}

resource "azurerm_network_security_group" "test1" {
    name = "acceptanceTestSecurityGroup2"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test1.name}"
}

resource "azurerm_network_security_rule" "test1" {
	name = "test123"
    	priority = 100
    	direction = "Outbound"
    	access = "Allow"
    	protocol = "Tcp"
    	source_port_range = "*"
    	destination_port_range = "*"
    	source_address_prefix = "*"
    	destination_address_prefix = "*"
    	resource_group_name = "${azurerm_resource_group.test1.name}"
    	network_security_group_name = "${azurerm_network_security_group.test1.name}"
}
`

var testAccAzureRMNetworkSecurityRule_updateExtraRule = `
resource "azurerm_resource_group" "test1" {
    name = "acceptanceTestResourceGroup2"
    location = "West US"
}

resource "azurerm_network_security_group" "test1" {
    name = "acceptanceTestSecurityGroup2"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test1.name}"
}

resource "azurerm_network_security_rule" "test1" {
	name = "test123"
    	priority = 100
    	direction = "Outbound"
    	access = "Allow"
    	protocol = "Tcp"
    	source_port_range = "*"
    	destination_port_range = "*"
    	source_address_prefix = "*"
    	destination_address_prefix = "*"
    	resource_group_name = "${azurerm_resource_group.test1.name}"
    	network_security_group_name = "${azurerm_network_security_group.test1.name}"
}

resource "azurerm_network_security_rule" "test2" {
	name = "testing456"
    	priority = 101
    	direction = "Inbound"
    	access = "Deny"
    	protocol = "Tcp"
    	source_port_range = "*"
    	destination_port_range = "*"
    	source_address_prefix = "*"
    	destination_address_prefix = "*"
    	resource_group_name = "${azurerm_resource_group.test1.name}"
    	network_security_group_name = "${azurerm_network_security_group.test1.name}"
}
`
