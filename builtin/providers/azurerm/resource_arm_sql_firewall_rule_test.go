package azurerm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/jen20/riviera/sql"
)

func TestAccAzureRMSqlFirewallRule_basic(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMSqlFirewallRule_basic, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMSqlFirewallRule_withUpdates, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSqlFirewallRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSqlFirewallRuleExists("azurerm_sql_firewall_rule.test"),
					resource.TestCheckResourceAttr("azurerm_sql_firewall_rule.test", "start_ip_address", "0.0.0.0"),
					resource.TestCheckResourceAttr("azurerm_sql_firewall_rule.test", "end_ip_address", "255.255.255.255"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSqlFirewallRuleExists("azurerm_sql_firewall_rule.test"),
					resource.TestCheckResourceAttr("azurerm_sql_firewall_rule.test", "start_ip_address", "10.0.17.62"),
					resource.TestCheckResourceAttr("azurerm_sql_firewall_rule.test", "end_ip_address", "10.0.17.62"),
				),
			},
		},
	})
}

func testCheckAzureRMSqlFirewallRuleExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).rivieraClient

		readRequest := conn.NewRequestForURI(rs.Primary.ID)
		readRequest.Command = &sql.GetFirewallRule{}

		readResponse, err := readRequest.Execute()
		if err != nil {
			return fmt.Errorf("Bad: GetFirewallRule: %s", err)
		}
		if !readResponse.IsSuccessful() {
			return fmt.Errorf("Bad: GetFirewallRule: %s", readResponse.Error)
		}

		return nil
	}
}

func testCheckAzureRMSqlFirewallRuleDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).rivieraClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_sql_firewall_rule" {
			continue
		}

		readRequest := conn.NewRequestForURI(rs.Primary.ID)
		readRequest.Command = &sql.GetFirewallRule{}

		readResponse, err := readRequest.Execute()
		if err != nil {
			return fmt.Errorf("Bad: GetFirewallRule: %s", err)
		}

		if readResponse.IsSuccessful() {
			return fmt.Errorf("Bad: SQL Server Firewall Rule still exists: %s", readResponse.Error)
		}
	}

	return nil
}

var testAccAzureRMSqlFirewallRule_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG_%d"
    location = "West US"
}
resource "azurerm_sql_server" "test" {
    name = "acctestsqlserver%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "West US"
    version = "12.0"
    administrator_login = "mradministrator"
    administrator_login_password = "thisIsDog11"
}

resource "azurerm_sql_firewall_rule" "test" {
    name = "acctestsqlserver%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    server_name = "${azurerm_sql_server.test.name}"
    start_ip_address = "0.0.0.0"
    end_ip_address = "255.255.255.255"
}
`

var testAccAzureRMSqlFirewallRule_withUpdates = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG_%d"
    location = "West US"
}
resource "azurerm_sql_server" "test" {
    name = "acctestsqlserver%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "West US"
    version = "12.0"
    administrator_login = "mradministrator"
    administrator_login_password = "thisIsDog11"
}

resource "azurerm_sql_firewall_rule" "test" {
    name = "acctestsqlserver%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    server_name = "${azurerm_sql_server.test.name}"
    start_ip_address = "10.0.17.62"
    end_ip_address = "10.0.17.62"
}
`
