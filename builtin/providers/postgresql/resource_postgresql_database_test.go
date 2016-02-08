package postgresql

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPostgresqlDatabase_Basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPostgresqlDatabaseDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccPostgresqlDatabaseConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPostgresqlDatabaseExists("postgresql_database.mydb", "myrole"),
					resource.TestCheckResourceAttr(
						"postgresql_database.mydb", "name", "mydb"),
					resource.TestCheckResourceAttr(
						"postgresql_database.mydb", "owner", "myrole"),
				),
			},
		},
	})
}

func TestAccPostgresqlDatabase_DefaultOwner(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPostgresqlDatabaseDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccPostgresqlDatabaseConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPostgresqlDatabaseExists("postgresql_database.mydb_default_owner", ""),
					resource.TestCheckResourceAttr(
						"postgresql_database.mydb_default_owner", "name", "mydb_default_owner"),
				),
			},
		},
	})
}

func testAccCheckPostgresqlDatabaseDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "postgresql_database" {
			continue
		}

		exists, err := checkDatabaseExists(client, rs.Primary.ID)

		if err != nil {
			return fmt.Errorf("Error checking db %s", err)
		}

		if exists {
			return fmt.Errorf("Db still exists after destroy")
		}
	}

	return nil
}

func testAccCheckPostgresqlDatabaseExists(n string, owner string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Resource not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		actualOwner := rs.Primary.Attributes["owner"]
		if actualOwner != owner {
			return fmt.Errorf("Wrong owner for db expected %s got %s", owner, actualOwner)
		}

		client := testAccProvider.Meta().(*Client)
		exists, err := checkDatabaseExists(client, rs.Primary.ID)

		if err != nil {
			return fmt.Errorf("Error checking db %s", err)
		}

		if !exists {
			return fmt.Errorf("Db not found")
		}

		return nil
	}
}

func checkDatabaseExists(client *Client, dbName string) (bool, error) {
	conn, err := client.Connect()
	if err != nil {
		return false, err
	}
	defer conn.Close()

	var _rez int
	err = conn.QueryRow("SELECT 1 from pg_database d WHERE datname=$1", dbName).Scan(&_rez)
	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, fmt.Errorf("Error reading info about database: %s", err)
	default:
		return true, nil
	}
}

var testAccPostgresqlDatabaseConfig = `
resource "postgresql_role" "myrole" {
  name = "myrole"
  login = true
}

resource "postgresql_database" "mydb" {
   name = "mydb"
   owner = "${postgresql_role.myrole.name}"
}

resource "postgresql_database" "mydb2" {
   name = "mydb2"
   owner = "${postgresql_role.myrole.name}"
}

resource "postgresql_database" "mydb_default_owner" {
   name = "mydb_default_owner"
}

`
