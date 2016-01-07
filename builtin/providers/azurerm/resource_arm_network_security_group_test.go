package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceAzureRMNetworkSecurityGroupProtocol_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "Random",
			ErrCount: 1,
		},
		{
			Value:    "tcp",
			ErrCount: 0,
		},
		{
			Value:    "TCP",
			ErrCount: 0,
		},
		{
			Value:    "*",
			ErrCount: 0,
		},
		{
			Value:    "Udp",
			ErrCount: 0,
		},
		{
			Value:    "Tcp",
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateNetworkSecurityRuleProtocol(tc.Value, "azurerm_network_security_group")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Network Security Group protocol to trigger a validation error")
		}
	}
}

func TestResourceAzureRMNetworkSecurityGroupAccess_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "Random",
			ErrCount: 1,
		},
		{
			Value:    "Allow",
			ErrCount: 0,
		},
		{
			Value:    "Deny",
			ErrCount: 0,
		},
		{
			Value:    "ALLOW",
			ErrCount: 0,
		},
		{
			Value:    "deny",
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateNetworkSecurityRuleAccess(tc.Value, "azurerm_network_security_group")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Network Security Group access to trigger a validation error")
		}
	}
}

func TestResourceAzureRMNetworkSecurityGroupDirection_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "Random",
			ErrCount: 1,
		},
		{
			Value:    "Inbound",
			ErrCount: 0,
		},
		{
			Value:    "Outbound",
			ErrCount: 0,
		},
		{
			Value:    "INBOUND",
			ErrCount: 0,
		},
		{
			Value:    "Inbound",
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateNetworkSecurityRuleDirection(tc.Value, "azurerm_network_security_group")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Network Security Group direction to trigger a validation error")
		}
	}
}

func TestAccAzureRMNetworkSecurityGroup_basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMNetworkSecurityGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureRMNetworkSecurityGroup_basic,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkSecurityGroupExists("azurerm_network_security_group.test"),
				),
			},
		},
	})
}

func TestAccAzureRMNetworkSecurityGroup_addingExtraRules(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMNetworkSecurityGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureRMNetworkSecurityGroup_basic,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkSecurityGroupExists("azurerm_network_security_group.test"),
					resource.TestCheckResourceAttr(
						"azurerm_network_security_group.test", "security_rule.#", "1"),
				),
			},

			resource.TestStep{
				Config: testAccAzureRMNetworkSecurityGroup_anotherRule,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkSecurityGroupExists("azurerm_network_security_group.test"),
					resource.TestCheckResourceAttr(
						"azurerm_network_security_group.test", "security_rule.#", "2"),
				),
			},
		},
	})
}

func testCheckAzureRMNetworkSecurityGroupExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		sgName := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for network security group: %s", sgName)
		}

		conn := testAccProvider.Meta().(*ArmClient).secGroupClient

		resp, err := conn.Get(resourceGroup, sgName)
		if err != nil {
			return fmt.Errorf("Bad: Get on secGroupClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Network Security Group %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMNetworkSecurityGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).secGroupClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_network_security_group" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Network Security Group still exists:\n%#v", resp.Properties)
		}
	}

	return nil
}

var testAccAzureRMNetworkSecurityGroup_basic = `
resource "azurerm_resource_group" "test" {
    name = "acceptanceTestResourceGroup1"
    location = "West US"
}

resource "azurerm_network_security_group" "test" {
    name = "acceptanceTestSecurityGroup1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    security_rule {
    	name = "test123"
    	priority = 100
    	direction = "Inbound"
    	access = "Allow"
    	protocol = "Tcp"
    	source_port_range = "*"
    	destination_port_range = "*"
    	source_address_prefix = "*"
    	destination_address_prefix = "*"
    }
}
`

var testAccAzureRMNetworkSecurityGroup_anotherRule = `
resource "azurerm_resource_group" "test" {
    name = "acceptanceTestResourceGroup1"
    location = "West US"
}

resource "azurerm_network_security_group" "test" {
    name = "acceptanceTestSecurityGroup1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    security_rule {
    	name = "test123"
    	priority = 100
    	direction = "Inbound"
    	access = "Allow"
    	protocol = "Tcp"
    	source_port_range = "*"
    	destination_port_range = "*"
    	source_address_prefix = "*"
    	destination_address_prefix = "*"
    }

    security_rule {
    	name = "testDeny"
    	priority = 101
    	direction = "Inbound"
    	access = "Deny"
    	protocol = "Udp"
    	source_port_range = "*"
    	destination_port_range = "*"
    	source_address_prefix = "*"
    	destination_address_prefix = "*"
    }
}
`
