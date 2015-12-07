package postgresql

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPostgresqlRole_Basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPostgresqlRoleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccPostgresqlRoleConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPostgresqlRoleExists("postgresql_role.myrole2", "true"),
					resource.TestCheckResourceAttr(
						"postgresql_role.myrole2", "name", "myrole2"),
					resource.TestCheckResourceAttr(
						"postgresql_role.myrole2", "login", "true"),
				),
			},
		},
	})
}

func testAccCheckPostgresqlRoleDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "postgresql_role" {
			continue
		}

		exists, err := checkRoleExists(client, rs.Primary.ID)

		if err != nil {
			return fmt.Errorf("Error checking role %s", err)
		}

		if exists {
			return fmt.Errorf("Role still exists after destroy")
		}
	}

	return nil
}

func testAccCheckPostgresqlRoleExists(n string, canLogin string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Resource not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		actualCanLogin := rs.Primary.Attributes["login"]
		if actualCanLogin != canLogin {
			return fmt.Errorf("Wrong value for login expected %s got %s", canLogin, actualCanLogin)
		}

		client := testAccProvider.Meta().(*Client)
		exists, err := checkRoleExists(client, rs.Primary.ID)

		if err != nil {
			return fmt.Errorf("Error checking role %s", err)
		}

		if !exists {
			return fmt.Errorf("Role not found")
		}

		return nil
	}
}

func checkRoleExists(client *Client, roleName string) (bool, error) {
	conn, err := client.Connect()
	if err != nil {
		return false, err
	}
	defer conn.Close()

	var _rez int
	err = conn.QueryRow("SELECT 1 from pg_roles d WHERE rolname=$1", roleName).Scan(&_rez)
	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, fmt.Errorf("Error reading info about role: %s", err)
	default:
		return true, nil
	}
}

var testAccPostgresqlRoleConfig = `
resource "postgresql_role" "myrole2" {
  name = "myrole2"
  login = true
}

resource "postgresql_role" "role_with_pwd" {
  name = "role_with_pwd"
  login = true
  password = "mypass"
}

resource "postgresql_role" "role_with_pwd_encr" {
  name = "role_with_pwd_encr"
  login = true
  password = "mypass"
  encrypted = true
}

resource "postgresql_role" "role_with_pwd_no_login" {
  name = "role_with_pwd_no_login"
  password = "mypass"
}

resource "postgresql_role" "role_simple" {
  name = "role_simple"
}
`
