package postgresql

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPostgresqlSchema_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPostgresqlSchemaDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPostgresqlSchemaConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPostgresqlSchemaExists("postgresql_schema.test1", "foo"),
					resource.TestCheckResourceAttr(
						"postgresql_role.myrole3", "name", "myrole3"),
					resource.TestCheckResourceAttr(
						"postgresql_role.myrole3", "login", "true"),

					resource.TestCheckResourceAttr(
						"postgresql_schema.test1", "name", "foo"),
					// `postgres` is a calculated value
					// based on the username used in the
					// provider
					resource.TestCheckResourceAttr(
						"postgresql_schema.test1", "authorization", "postgres"),
				),
			},
		},
	})

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPostgresqlSchemaDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPostgresqlSchemaAuthConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPostgresqlSchemaExists("postgresql_schema.test2", "foo2"),
					resource.TestCheckResourceAttr(
						"postgresql_role.myrole4", "name", "myrole4"),
					resource.TestCheckResourceAttr(
						"postgresql_role.myrole4", "login", "true"),

					resource.TestCheckResourceAttr(
						"postgresql_schema.test2", "name", "foo2"),
					resource.TestCheckResourceAttr(
						"postgresql_schema.test2", "authorization", "myrole4"),
				),
			},
		},
	})
}

func testAccCheckPostgresqlSchemaDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "postgresql_schema" {
			continue
		}

		exists, err := checkSchemaExists(client, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error checking schema %s", err)
		}

		if exists {
			return fmt.Errorf("Schema still exists after destroy")
		}
	}

	return nil
}

func testAccCheckPostgresqlSchemaExists(n string, schemaName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Resource not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		actualSchemaName := rs.Primary.Attributes["name"]
		if actualSchemaName != schemaName {
			return fmt.Errorf("Wrong value for schema name expected %s got %s", schemaName, actualSchemaName)
		}

		client := testAccProvider.Meta().(*Client)
		exists, err := checkSchemaExists(client, rs.Primary.ID)

		if err != nil {
			return fmt.Errorf("Error checking schema %s", err)
		}

		if !exists {
			return fmt.Errorf("Schema not found")
		}

		return nil
	}
}

func checkSchemaExists(client *Client, schemaName string) (bool, error) {
	conn, err := client.Connect()
	if err != nil {
		return false, err
	}
	defer conn.Close()

	var _rez string
	err = conn.QueryRow("SELECT nspname FROM pg_catalog.pg_namespace WHERE nspname=$1", schemaName).Scan(&_rez)
	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, fmt.Errorf("Error reading info about schema: %s", err)
	default:
		return true, nil
	}
}

var testAccPostgresqlSchemaConfig = `
resource "postgresql_role" "myrole3" {
  name = "myrole3"
  login = true
}

resource "postgresql_schema" "test1" {
  name = "foo"
}
`

var testAccPostgresqlSchemaAuthConfig = `
resource "postgresql_role" "myrole4" {
  name = "myrole4"
  login = true
}

resource "postgresql_schema" "test2" {
  name = "foo2"
  authorization = "${postgresql_role.myrole4.name}"
}
`
