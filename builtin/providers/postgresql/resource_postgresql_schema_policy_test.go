package postgresql

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPostgreSQLSchemaPolicy_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPostgreSQLSchemaPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPostgreSQLSchemaPolicyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPostgreSQLSchemaPolicyExists("postgresql_schema.test1", "foo"),
					resource.TestCheckResourceAttr(
						"postgresql_role.dba", "name", "dba"),
					resource.TestCheckResourceAttr(
						"postgresql_role.app1", "name", "app1"),
					resource.TestCheckResourceAttr(
						"postgresql_role.app2", "name", "app2"),

					resource.TestCheckResourceAttr(
						"postgresql_schema.foo", "name", "foo"),
					resource.TestCheckResourceAttr(
						"postgresql_schema.owner", "name", "dba"),

					resource.TestCheckResourceAttr(
						"postgresql_schema_policy.foo_allow", "create", "false"),
					resource.TestCheckResourceAttr(
						"postgresql_schema_policy.foo_allow", "create_with_grant", "true"),
					resource.TestCheckResourceAttr(
						"postgresql_schema_policy.foo_allow", "usage", "false"),
					resource.TestCheckResourceAttr(
						"postgresql_schema_policy.foo_allow", "usage_with_grant", "true"),
					resource.TestCheckResourceAttr(
						"postgresql_schema_policy.foo_allow", "schema", "foo"),
					resource.TestCheckResourceAttr(
						"postgresql_schema_policy.foo_allow", "role", "app1"),

					resource.TestCheckResourceAttr(
						"postgresql_schema_policy.foo_deny", "schema", "foo"),
					resource.TestCheckResourceAttr(
						"postgresql_schema_policy.foo_deny", "role", "app2"),
				),
			},
		},
	})
}

func testAccCheckPostgreSQLSchemaPolicyDestroy(s *terraform.State) error {
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

func testAccCheckPostgreSQLSchemaPolicyExists(n string, schemaName string) resource.TestCheckFunc {
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

func checkSchemaPolicyExists(client *Client, schemaName string) (bool, error) {
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

var testAccPostgreSQLSchemaPolicyConfig = `
resource "postgresql_role" "dba" {
  name = "dba"
}

resource "postgresql_role" "app1" {
  name = "app1"
  # depends_on = ["postgresql_schema_policy.foo_allow"]
}

resource "postgresql_role" "app2" {
  name = "app2"
}

resource "postgresql_schema" "foo" {
  name = "foo"
  owner = "${postgresql_role.dba.name}"
}

resource "postgresql_schema_policy" "foo_allow" {
  create_with_grant = true
  usage_with_grant = true

  schema = "${postgresql_schema.foo.name}"
  role = "${postgresql_role.app1.name}"
}

resource "postgresql_schema_policy" "foo_deny" {
  schema = "${postgresql_schema.foo.name}"
  role = "${postgresql_role.app2.name}"
}
`
