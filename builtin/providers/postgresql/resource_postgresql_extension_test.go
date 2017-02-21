package postgresql

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPostgresqlExtension_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPostgresqlExtensionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPostgresqlExtensionConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPostgresqlExtensionExists("postgresql_extension.myextension"),
					resource.TestCheckResourceAttr(
						"postgresql_extension.myextension", "name", "pg_trgm"),
					resource.TestCheckResourceAttr(
						"postgresql_extension.myextension", "schema", "public"),

					// NOTE(sean): Version 1.3 is what's
					// shipped with PostgreSQL 9.6.1.  This
					// version number may drift in the
					// future.
					resource.TestCheckResourceAttr(
						"postgresql_extension.myextension", "version", "1.3"),
				),
			},
		},
	})
}

func testAccCheckPostgresqlExtensionDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "postgresql_extension" {
			continue
		}

		exists, err := checkExtensionExists(client, rs.Primary.ID)

		if err != nil {
			return fmt.Errorf("Error checking extension %s", err)
		}

		if exists {
			return fmt.Errorf("Extension still exists after destroy")
		}
	}

	return nil
}

func testAccCheckPostgresqlExtensionExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Resource not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*Client)
		exists, err := checkExtensionExists(client, rs.Primary.ID)

		if err != nil {
			return fmt.Errorf("Error checking extension %s", err)
		}

		if !exists {
			return fmt.Errorf("Extension not found")
		}

		return nil
	}
}

func TestAccPostgresqlExtension_SchemaRename(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPostgresqlExtensionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPostgresqlExtensionSchemaChange1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPostgresqlExtensionExists("postgresql_extension.ext1trgm"),
					resource.TestCheckResourceAttr(
						"postgresql_schema.ext1foo", "name", "foo"),
					resource.TestCheckResourceAttr(
						"postgresql_extension.ext1trgm", "name", "pg_trgm"),
					resource.TestCheckResourceAttr(
						"postgresql_extension.ext1trgm", "name", "pg_trgm"),
					resource.TestCheckResourceAttr(
						"postgresql_extension.ext1trgm", "schema", "foo"),
				),
			},
			{
				Config: testAccPostgresqlExtensionSchemaChange2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPostgresqlExtensionExists("postgresql_extension.ext1trgm"),
					resource.TestCheckResourceAttr(
						"postgresql_schema.ext1foo", "name", "bar"),
					resource.TestCheckResourceAttr(
						"postgresql_extension.ext1trgm", "name", "pg_trgm"),
					resource.TestCheckResourceAttr(
						"postgresql_extension.ext1trgm", "schema", "bar"),
				),
			},
		},
	})
}

func checkExtensionExists(client *Client, extensionName string) (bool, error) {
	conn, err := client.Connect()
	if err != nil {
		return false, err
	}
	defer conn.Close()

	var _rez bool
	err = conn.QueryRow("SELECT TRUE from pg_catalog.pg_extension d WHERE extname=$1", extensionName).Scan(&_rez)
	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, fmt.Errorf("Error reading info about extension: %s", err)
	default:
		return true, nil
	}
}

var testAccPostgresqlExtensionConfig = `
resource "postgresql_extension" "myextension" {
  name = "pg_trgm"
}
`

var testAccPostgresqlExtensionSchemaChange1 = `
resource "postgresql_schema" "ext1foo" {
  name = "foo"
}

resource "postgresql_extension" "ext1trgm" {
  name = "pg_trgm"
  schema = "${postgresql_schema.ext1foo.name}"
}
`

var testAccPostgresqlExtensionSchemaChange2 = `
resource "postgresql_schema" "ext1foo" {
  name = "bar"
}

resource "postgresql_extension" "ext1trgm" {
  name = "pg_trgm"
  schema = "${postgresql_schema.ext1foo.name}"
}
`
