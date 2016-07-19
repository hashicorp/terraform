package mssql

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccMSsqlDatabase_Basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMSsqlDatabaseDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccMSsqlDatabaseConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMSsqlDatabaseExists("mssql_database.mydb"),
					resource.TestCheckResourceAttr(
						"mssql_database.mydb", "name", "mydb"),
				),
			},
		},
	})
}

func testAccCheckMSsqlDatabaseDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "mssql_database" {
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

func testAccCheckMSsqlDatabaseExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Resource not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
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

	result, err := conn.Query("SELECT db_id('" + dbName + "')")
	if err != nil {
		return false, nil
	}

	for result.Next() {
		var s sql.NullString
		err := result.Scan(&s)
		if err != nil {
			return false, nil
		}

		// Check result
		if s.Valid {
			return true, nil
		}
		return false, nil
	}

	return false, nil
}

var testAccMSsqlDatabaseConfig = `
resource "mssql_database" "mydb" {
   name = "mydb"
}

resource "mssql_database" "mydb2" {
   name = "mydb2"
}
`
