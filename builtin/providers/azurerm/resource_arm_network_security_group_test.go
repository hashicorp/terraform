package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMNetworkSecurityGroup_basic(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMNetworkSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMNetworkSecurityGroup_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkSecurityGroupExists("azurerm_network_security_group.test"),
				),
			},
		},
	})
}

func TestAccAzureRMNetworkSecurityGroup_disappears(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMNetworkSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMNetworkSecurityGroup_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkSecurityGroupExists("azurerm_network_security_group.test"),
					testCheckAzureRMNetworkSecurityGroupDisappears("azurerm_network_security_group.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAzureRMNetworkSecurityGroup_withTags(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMNetworkSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMNetworkSecurityGroup_withTags(rInt),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkSecurityGroupExists("azurerm_network_security_group.test"),
					resource.TestCheckResourceAttr(
						"azurerm_network_security_group.test", "tags.%", "2"),
					resource.TestCheckResourceAttr(
						"azurerm_network_security_group.test", "tags.environment", "Production"),
					resource.TestCheckResourceAttr(
						"azurerm_network_security_group.test", "tags.cost_center", "MSFT"),
				),
			},

			{
				Config: testAccAzureRMNetworkSecurityGroup_withTagsUpdate(rInt),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkSecurityGroupExists("azurerm_network_security_group.test"),
					resource.TestCheckResourceAttr(
						"azurerm_network_security_group.test", "tags.%", "1"),
					resource.TestCheckResourceAttr(
						"azurerm_network_security_group.test", "tags.environment", "staging"),
				),
			},
		},
	})
}

func TestAccAzureRMNetworkSecurityGroup_addingExtraRules(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMNetworkSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMNetworkSecurityGroup_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkSecurityGroupExists("azurerm_network_security_group.test"),
					resource.TestCheckResourceAttr(
						"azurerm_network_security_group.test", "security_rule.#", "1"),
				),
			},

			{
				Config: testAccAzureRMNetworkSecurityGroup_anotherRule(rInt),
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

		resp, err := conn.Get(resourceGroup, sgName, "")
		if err != nil {
			return fmt.Errorf("Bad: Get on secGroupClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Network Security Group %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMNetworkSecurityGroupDisappears(name string) resource.TestCheckFunc {
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

		_, error := conn.Delete(resourceGroup, sgName, make(chan struct{}))
		err := <-error
		if err != nil {
			return fmt.Errorf("Bad: Delete on secGroupClient: %s", err)
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

		resp, err := conn.Get(resourceGroup, name, "")

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Network Security Group still exists:\n%#v", resp.SecurityGroupPropertiesFormat)
		}
	}

	return nil
}

func testAccAzureRMNetworkSecurityGroup_basic(rInt int) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
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
    	protocol = "TCP"
    	source_port_range = "*"
    	destination_port_range = "*"
    	source_address_prefix = "*"
    	destination_address_prefix = "*"
    }
}
`, rInt)
}

func testAccAzureRMNetworkSecurityGroup_anotherRule(rInt int) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
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
`, rInt)
}

func testAccAzureRMNetworkSecurityGroup_withTags(rInt int) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
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


    tags {
	environment = "Production"
	cost_center = "MSFT"
    }
}
`, rInt)
}

func testAccAzureRMNetworkSecurityGroup_withTagsUpdate(rInt int) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
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

    tags {
	environment = "staging"
    }
}
`, rInt)
}
