package azure

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureSqlDatabaseServiceBasic(t *testing.T) {
	name := "azure_sql_database_service.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureSqlDatabaseServiceDeleted,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureSqlDatabaseServiceConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccAzureSqlDatabaseServerGetName,
					testAccCheckAzureSqlDatabaseServiceExists(name),
					resource.TestCheckResourceAttr(name, "name", "terraform-testing-db"),
					resource.TestCheckResourceAttrPtr(name, "database_server_name",
						testAccAzureSqlServerName),
					resource.TestCheckResourceAttr(name, "collation",
						"SQL_Latin1_General_CP1_CI_AS"),
					resource.TestCheckResourceAttr(name, "edition", "Standard"),
				),
			},
		},
	})
}

func TestAccAzureSqlDatabaseServiceAdvanced(t *testing.T) {
	name := "azure_sql_database_service.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureSqlDatabaseServiceDeleted,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureSqlDatabaseServiceConfigAdvanced,
				Check: resource.ComposeTestCheckFunc(
					testAccAzureSqlDatabaseServerGetName,
					testAccCheckAzureSqlDatabaseServiceExists(name),
					resource.TestCheckResourceAttr(name, "name", "terraform-testing-db"),
					resource.TestCheckResourceAttrPtr(name, "database_server_name",
						testAccAzureSqlServerName),
					resource.TestCheckResourceAttr(name, "edition", "Premium"),
					resource.TestCheckResourceAttr(name, "collation",
						"Arabic_BIN"),
					resource.TestCheckResourceAttr(name, "max_size_bytes", "10737418240"),
					resource.TestCheckResourceAttr(name, "service_level_id",
						"7203483a-c4fb-4304-9e9f-17c71c904f5d"),
				),
			},
		},
	})
}

func TestAccAzureSqlDatabaseServiceUpdate(t *testing.T) {
	name := "azure_sql_database_service.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureSqlDatabaseServiceDeleted,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureSqlDatabaseServiceConfigAdvanced,
				Check: resource.ComposeTestCheckFunc(
					testAccAzureSqlDatabaseServerGetName,
					testAccCheckAzureSqlDatabaseServiceExists(name),
					resource.TestCheckResourceAttr(name, "name", "terraform-testing-db"),
					resource.TestCheckResourceAttrPtr(name, "database_server_name",
						testAccAzureSqlServerName),
					resource.TestCheckResourceAttr(name, "edition", "Premium"),
					resource.TestCheckResourceAttr(name, "collation",
						"Arabic_BIN"),
					resource.TestCheckResourceAttr(name, "max_size_bytes", "10737418240"),
					resource.TestCheckResourceAttr(name, "service_level_id",
						"7203483a-c4fb-4304-9e9f-17c71c904f5d"),
				),
			},
			resource.TestStep{
				Config: testAccAzureSqlDatabaseServiceConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccAzureSqlDatabaseServerGetName,
					testAccCheckAzureSqlDatabaseServiceExists(name),
					resource.TestCheckResourceAttr(name, "name",
						"terraform-testing-db-renamed"),
					resource.TestCheckResourceAttrPtr(name, "database_server_name",
						testAccAzureSqlServerName),
					resource.TestCheckResourceAttr(name, "edition", "Standard"),
					resource.TestCheckResourceAttr(name, "collation",
						"SQL_Latin1_General_CP1_CI_AS"),
					resource.TestCheckResourceAttr(name, "max_size_bytes", "5368709120"),
					resource.TestCheckResourceAttr(name, "service_level_id",
						"f1173c43-91bd-4aaa-973c-54e79e15235b"),
				),
			},
		},
	})
}

func testAccCheckAzureSqlDatabaseServiceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("SQL Service %s doesn't exist.", name)
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("SQL Service %s resource ID not set.", name)
		}

		sqlClient := testAccProvider.Meta().(*Client).sqlClient
		dbs, err := sqlClient.ListDatabases(*testAccAzureSqlServerName)
		if err != nil {
			return fmt.Errorf("Error issuing Azure SQL Service list request: %s", err)
		}

		for _, srv := range dbs.ServiceResources {
			if srv.Name == resource.Primary.ID {
				return nil
			}
		}

		return fmt.Errorf("SQL Service %s doesn't exist.", name)
	}
}

func testAccCheckAzureSqlDatabaseServiceDeleted(s *terraform.State) error {
	for _, resource := range s.RootModule().Resources {
		if resource.Type != "azure_sql_database_service" {
			continue
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("SQL Service resource ID not set.")
		}

		sqlClient := testAccProvider.Meta().(*Client).sqlClient
		dbs, err := sqlClient.ListDatabases(*testAccAzureSqlServerName)
		if err != nil {
			// ¯\_(ツ)_/¯
			if strings.Contains(err.Error(), "Cannot open server") {
				return nil
			}
			return fmt.Errorf("Error issuing Azure SQL Service list request: %s", err)
		}

		for _, srv := range dbs.ServiceResources {
			if srv.Name == resource.Primary.ID {
				return fmt.Errorf("SQL Service %s still exists.", resource.Primary.ID)
			}
		}
	}

	return nil
}

const testAccAzureSqlDatabaseServiceConfigBasic = testAccAzureSqlDatabaseServerConfig + `
resource "azure_sql_database_service" "foo" {
    name = "terraform-testing-db"
    database_server_name = "${azure_sql_database_server.foo.name}"
    edition = "Standard"
}
`

const testAccAzureSqlDatabaseServiceConfigAdvanced = testAccAzureSqlDatabaseServerConfig + `
resource "azure_sql_database_service" "foo" {
    name = "terraform-testing-db"
    database_server_name = "${azure_sql_database_server.foo.name}"
    edition = "Premium"
    collation = "Arabic_BIN"
    max_size_bytes = "10737418240"
    service_level_id = "7203483a-c4fb-4304-9e9f-17c71c904f5d"
}
`

const testAccAzureSqlDatabaseServiceConfigUpdate = testAccAzureSqlDatabaseServerConfig + `
resource "azure_sql_database_service" "foo" {
    name = "terraform-testing-db-renamed"
    database_server_name = "${azure_sql_database_server.foo.name}"
    edition = "Standard"
    collation = "SQL_Latin1_General_CP1_CI_AS"
    max_size_bytes = "5368709120"
    service_level_id = "f1173c43-91bd-4aaa-973c-54e79e15235b"
}
`
