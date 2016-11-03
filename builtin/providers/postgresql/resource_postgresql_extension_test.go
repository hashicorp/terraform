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

func checkExtensionExists(client *Client, extensionName string) (bool, error) {
	conn, err := client.Connect()
	if err != nil {
		return false, err
	}
	defer conn.Close()

	var _rez int
	err = conn.QueryRow("SELECT 1 from pg_extension d WHERE extname=$1", extensionName).Scan(&_rez)
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
