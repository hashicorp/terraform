package azure

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/management/sql"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureSqlDatabaseServerFirewallRuleBasic(t *testing.T) {
	name := "azure_sql_database_server_firewall_rule.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccAzureDatabaseServerFirewallRuleDeleted(testAccAzureSqlServerNames),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureDatabaseServerFirewallRuleBasicConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccAzureSqlDatabaseServerGetNames,
					testAccAzureSqlDatabaseServersNumber(1),
					testAccAzureDatabaseServerFirewallRuleExists(name, testAccAzureSqlServerNames),
					resource.TestCheckResourceAttr(name, "name", "terraform-testing-rule"),
					resource.TestCheckResourceAttr(name, "start_ip", "10.0.0.0"),
					resource.TestCheckResourceAttr(name, "end_ip", "10.0.0.255"),
				),
			},
		},
	})
}

func TestAccAzureSqlDatabaseServerFirewallRuleAdvanced(t *testing.T) {
	name1 := "azure_sql_database_server_firewall_rule.foo"
	name2 := "azure_sql_database_server_firewall_rule.bar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccAzureDatabaseServerFirewallRuleDeleted(testAccAzureSqlServerNames),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureDatabaseServerFirewallRuleAdvancedConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccAzureSqlDatabaseServerGetNames,
					testAccAzureSqlDatabaseServersNumber(2),
					//testAccAzureDatabaseServerFirewallRuleExists(name1, testAccAzureSqlServerNames),
					resource.TestCheckResourceAttr(name1, "name", "terraform-testing-rule1"),
					resource.TestCheckResourceAttr(name1, "start_ip", "10.0.0.0"),
					resource.TestCheckResourceAttr(name1, "end_ip", "10.0.0.255"),
					//testAccAzureDatabaseServerFirewallRuleExists(name2, testAccAzureSqlServerNames),
					resource.TestCheckResourceAttr(name2, "name", "terraform-testing-rule2"),
					resource.TestCheckResourceAttr(name2, "start_ip", "200.0.0.0"),
					resource.TestCheckResourceAttr(name2, "end_ip", "200.255.255.255"),
				),
			},
		},
	})
}

func TestAccAzureSqlDatabaseServerFirewallRuleUpdate(t *testing.T) {
	name1 := "azure_sql_database_server_firewall_rule.foo"
	name2 := "azure_sql_database_server_firewall_rule.bar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccAzureDatabaseServerFirewallRuleDeleted(testAccAzureSqlServerNames),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureDatabaseServerFirewallRuleAdvancedConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccAzureSqlDatabaseServerGetNames,
					testAccAzureSqlDatabaseServersNumber(2),
					//testAccAzureDatabaseServerFirewallRuleExists(name1, testAccAzureSqlServerNames),
					resource.TestCheckResourceAttr(name1, "name", "terraform-testing-rule1"),
					resource.TestCheckResourceAttr(name1, "start_ip", "10.0.0.0"),
					resource.TestCheckResourceAttr(name1, "end_ip", "10.0.0.255"),
					//testAccAzureDatabaseServerFirewallRuleExists(name2, testAccAzureSqlServerNames),
					resource.TestCheckResourceAttr(name2, "name", "terraform-testing-rule2"),
					resource.TestCheckResourceAttr(name2, "start_ip", "200.0.0.0"),
					resource.TestCheckResourceAttr(name2, "end_ip", "200.255.255.255"),
				),
			},
			resource.TestStep{
				Config: testAccAzureDatabaseServerFirewallRuleUpdateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccAzureSqlDatabaseServerGetNames,
					testAccAzureSqlDatabaseServersNumber(2),
					//testAccAzureDatabaseServerFirewallRuleExists(name1, testAccAzureSqlServerNames),
					resource.TestCheckResourceAttr(name1, "name", "terraform-testing-rule1"),
					resource.TestCheckResourceAttr(name1, "start_ip", "11.0.0.0"),
					resource.TestCheckResourceAttr(name1, "end_ip", "11.0.0.255"),
				),
			},
		},
	})
}

func testAccAzureDatabaseServerFirewallRuleExists(name string, servers []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		res, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Azure Database Server Firewall Rule %q doesn't exist.", name)
		}

		if res.Primary.ID == "" {
			return fmt.Errorf("Azure Database Server Firewall Rule %q res ID not set.", name)
		}

		sqlClient := testAccProvider.Meta().(*Client).sqlClient

		for _, server := range servers {
			var rules sql.ListFirewallRulesResponse

			err := resource.Retry(15*time.Minute, func() error {
				var erri error
				rules, erri = sqlClient.ListFirewallRules(server)
				if erri != nil {
					return fmt.Errorf("Error listing Azure Database Server Firewall Rules for Server %q: %s", server, erri)
				}

				return nil
			})
			if err != nil {
				return err
			}

			var found bool
			for _, rule := range rules.FirewallRules {
				if rule.Name == res.Primary.ID {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("Azure Database Server Firewall Rule %q doesn't exists on server %q.", res.Primary.ID, server)
			}
		}

		return nil
	}
}

func testAccAzureDatabaseServerFirewallRuleDeleted(servers []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, resource := range s.RootModule().Resources {
			if resource.Type != "azure_sql_database_server_firewall_rule" {
				continue
			}

			if resource.Primary.ID == "" {
				return fmt.Errorf("Azure Database Server Firewall Rule resource ID not set.")
			}

			sqlClient := testAccProvider.Meta().(*Client).sqlClient

			for _, server := range servers {
				rules, err := sqlClient.ListFirewallRules(server)
				if err != nil {
					// ¯\_(ツ)_/¯
					if strings.Contains(err.Error(), "Cannot open server") {
						return nil
					}
					return fmt.Errorf("Error listing Azure Database Server Firewall Rules for Server %q: %s", server, err)
				}

				for _, rule := range rules.FirewallRules {
					if rule.Name == resource.Primary.ID {
						return fmt.Errorf("Azure Database Server Firewall Rule %q still exists on Server %q.", resource.Primary.ID, err)
					}
				}
			}
		}

		return nil
	}
}

var testAccAzureDatabaseServerFirewallRuleBasicConfig = `
resource "azure_sql_database_server" "foo" {
	location = "West US"
	username = "SuperUser"
	password = "SuperSEKR3T"
	version = "2.0"
}

resource "azure_sql_database_server_firewall_rule" "foo" {
	name = "terraform-testing-rule"
	depends_on = ["azure_sql_database_server.foo"]
	start_ip = "10.0.0.0"
	end_ip = "10.0.0.255"
	database_server_names = ["${azure_sql_database_server.foo.name}"]
}
`

var testAccAzureDatabaseServerFirewallRuleAdvancedConfig = `
resource "azure_sql_database_server" "foo" {
	location = "West US"
	username = "SuperUser"
	password = "SuperSEKR3T"
	version = "2.0"
}

resource "azure_sql_database_server" "bar" {
	location = "West US"
	username = "SuperUser"
	password = "SuperSEKR3T"
	version = "2.0"
}

resource "azure_sql_database_server_firewall_rule" "foo" {
	name = "terraform-testing-rule1"
	start_ip = "10.0.0.0"
	end_ip = "10.0.0.255"
	database_server_names = ["${azure_sql_database_server.foo.name}", "${azure_sql_database_server.bar.name}"]
}

resource "azure_sql_database_server_firewall_rule" "bar" {
	name = "terraform-testing-rule2"
	start_ip = "200.0.0.0"
	end_ip = "200.255.255.255"
	database_server_names = ["${azure_sql_database_server.foo.name}", "${azure_sql_database_server.bar.name}"]
}
`

var testAccAzureDatabaseServerFirewallRuleUpdateConfig = `
resource "azure_sql_database_server" "foo" {
	location = "West US"
	username = "SuperUser"
	password = "SuperSEKR3T"
	version = "2.0"
}

resource "azure_sql_database_server" "bar" {
	location = "West US"
	username = "SuperUser"
	password = "SuperSEKR3T"
	version = "2.0"
}

resource "azure_sql_database_server_firewall_rule" "foo" {
	name = "terraform-testing-rule1"
	start_ip = "11.0.0.0"
	end_ip = "11.0.0.255"
	database_server_names = ["${azure_sql_database_server.foo.name}"]
}
`
