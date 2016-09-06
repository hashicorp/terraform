package postgresql

import (
	"database/sql"
	"fmt"
	"testing"

	"errors"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPostgresqlDatabase_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPostgresqlDatabaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPostgreSQLDatabaseConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPostgresqlDatabaseExists("postgresql_database.mydb"),
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
			{
				Config: testAccPostgreSQLDatabaseConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPostgresqlDatabaseExists("postgresql_database.mydb_default_owner"),
					resource.TestCheckResourceAttr(
						"postgresql_database.mydb_default_owner", "name", "mydb_default_owner"),
					resource.TestCheckResourceAttrSet(
						"postgresql_database.mydb_default_owner", "owner"),
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
			return errors.New("Db still exists after destroy")
		}
	}

	return nil
}

func testAccCheckPostgresqlDatabaseExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Resource not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("No ID is set")
		}

		client := testAccProvider.Meta().(*Client)
		exists, err := checkDatabaseExists(client, rs.Primary.ID)

		if err != nil {
			return fmt.Errorf("Error checking db %s", err)
		}

		if !exists {
			return errors.New("Db not found")
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

var testAccPostgreSQLDatabaseConfig = `
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

resource "postgresql_database" "mydb3" {
   name = "mydb3"
   owner = "${postgresql_role.myrole.name}"
   template = "template1"
   encoding = "SQL_ASCII"
   lc_collate = "C"
   lc_ctype = "C"
   tablespace_name = "pg_default"
   connection_limit = -1
   allow_connections = false
   is_template = false
}

resource "postgresql_database" "mydb_default_owner" {
   name = "mydb_default_owner"
}

`
