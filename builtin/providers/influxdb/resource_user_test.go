package influxdb

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/influxdata/influxdb/client"
)

func TestAccInfluxDBUser_admin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccUserConfig_admin,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserExists("influxdb_user.test"),
					resource.TestCheckResourceAttr(
						"influxdb_user.test", "name", "terraform_test",
					),
					resource.TestCheckResourceAttr(
						"influxdb_user.test", "password", "terraform",
					),
					resource.TestCheckResourceAttr(
						"influxdb_user.test", "admin", "true",
					),
				),
			},
			resource.TestStep{
				Config: testAccUserConfig_revoke,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserExists("influxdb_user.test"),
					testAccCheckUserNoAdmin("influxdb_user.test"),
					resource.TestCheckResourceAttr(
						"influxdb_user.test", "name", "terraform_test",
					),
					resource.TestCheckResourceAttr(
						"influxdb_user.test", "password", "terraform",
					),
					resource.TestCheckResourceAttr(
						"influxdb_user.test", "admin", "false",
					),
				),
			},
		},
	})
}

func TestAccInfluxDBUser_grant(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccUserConfig_grant,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserExists("influxdb_user.test"),
					testAccCheckUserGrants("influxdb_user.test", "terraform-green", "READ"),
					resource.TestCheckResourceAttr(
						"influxdb_user.test", "name", "terraform_test",
					),
					resource.TestCheckResourceAttr(
						"influxdb_user.test", "password", "terraform",
					),
					resource.TestCheckResourceAttr(
						"influxdb_user.test", "admin", "false",
					),
					resource.TestCheckResourceAttr(
						"influxdb_user.test", "grant.#", "1",
					),
				),
			},
			resource.TestStep{
				Config: testAccUserConfig_grantUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserGrants("influxdb_user.test", "terraform-green", "WRITE"),
					testAccCheckUserGrants("influxdb_user.test", "terraform-blue", "READ"),
					resource.TestCheckResourceAttr(
						"influxdb_user.test", "name", "terraform_test",
					),
					resource.TestCheckResourceAttr(
						"influxdb_user.test", "password", "terraform",
					),
					resource.TestCheckResourceAttr(
						"influxdb_user.test", "admin", "false",
					),
					resource.TestCheckResourceAttr(
						"influxdb_user.test", "grant.#", "2",
					),
				),
			},
		},
	})
}

func TestAccInfluxDBUser_revoke(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccUserConfig_grant,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserExists("influxdb_user.test"),
					testAccCheckUserGrants("influxdb_user.test", "terraform-green", "READ"),
					resource.TestCheckResourceAttr(
						"influxdb_user.test", "name", "terraform_test",
					),
					resource.TestCheckResourceAttr(
						"influxdb_user.test", "password", "terraform",
					),
					resource.TestCheckResourceAttr(
						"influxdb_user.test", "admin", "false",
					),
					resource.TestCheckResourceAttr(
						"influxdb_user.test", "grant.#", "1",
					),
				),
			},
			resource.TestStep{
				Config: testAccUserConfig_revoke,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserGrantsEmpty("influxdb_user.test"),
					resource.TestCheckResourceAttr(
						"influxdb_user.test", "name", "terraform_test",
					),
					resource.TestCheckResourceAttr(
						"influxdb_user.test", "password", "terraform",
					),
					resource.TestCheckResourceAttr(
						"influxdb_user.test", "admin", "false",
					),
					resource.TestCheckResourceAttr(
						"influxdb_user.test", "grant.#", "0",
					),
				),
			},
		},
	})
}

func testAccCheckUserExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No user id set")
		}

		conn := testAccProvider.Meta().(*client.Client)

		query := client.Query{
			Command: "SHOW USERS",
		}

		resp, err := conn.Query(query)
		if err != nil {
			return err
		}

		if resp.Err != nil {
			return resp.Err
		}

		for _, result := range resp.Results[0].Series[0].Values {
			if result[0] == rs.Primary.Attributes["name"] {
				return nil
			}
		}

		return fmt.Errorf("User %q does not exist", rs.Primary.Attributes["name"])
	}
}

func testAccCheckUserNoAdmin(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No user id set")
		}

		conn := testAccProvider.Meta().(*client.Client)

		query := client.Query{
			Command: "SHOW USERS",
		}

		resp, err := conn.Query(query)
		if err != nil {
			return err
		}

		if resp.Err != nil {
			return resp.Err
		}

		for _, result := range resp.Results[0].Series[0].Values {
			if result[0] == rs.Primary.Attributes["name"] {
				if result[1].(bool) == true {
					return fmt.Errorf("User %q is admin", rs.Primary.ID)
				}

				return nil
			}
		}

		return fmt.Errorf("User %q does not exist", rs.Primary.Attributes["name"])
	}
}

func testAccCheckUserGrantsEmpty(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No user id set")
		}

		conn := testAccProvider.Meta().(*client.Client)

		query := client.Query{
			Command: fmt.Sprintf("SHOW GRANTS FOR %s", rs.Primary.Attributes["name"]),
		}

		resp, err := conn.Query(query)
		if err != nil {
			return err
		}

		if resp.Err != nil {
			return resp.Err
		}

		for _, result := range resp.Results[0].Series[0].Values {
			if result[1].(string) != "NO PRIVILEGES" {
				return fmt.Errorf("User %q still has grants: %#v", rs.Primary.ID, resp.Results[0].Series[0].Values)
			}
		}

		return nil
	}
}

func testAccCheckUserGrants(n, database, privilege string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No user id set")
		}

		conn := testAccProvider.Meta().(*client.Client)

		query := client.Query{
			Command: fmt.Sprintf("SHOW GRANTS FOR %s", rs.Primary.Attributes["name"]),
		}

		resp, err := conn.Query(query)
		if err != nil {
			return err
		}

		if resp.Err != nil {
			return resp.Err
		}

		for _, result := range resp.Results[0].Series[0].Values {
			if result[0].(string) == database && result[1].(string) == privilege {
				return nil
			}
		}

		return fmt.Errorf("Privilege %q on %q for %q does not exist", privilege, database, rs.Primary.Attributes["name"])
	}
}

var testAccUserConfig_admin = `
resource "influxdb_user" "test" {
    name = "terraform_test"
    password = "terraform"
    admin = true
}
`

var testAccUserConfig_grant = `
resource "influxdb_database" "green" {
    name = "terraform-green"
}

resource "influxdb_user" "test" {
    name = "terraform_test"
    password = "terraform"

    grant {
      database = "${influxdb_database.green.name}"
      privilege = "read"
    }
}
`

var testAccUserConfig_revoke = `
resource "influxdb_database" "green" {
    name = "terraform-green"
}

resource "influxdb_user" "test" {
    name = "terraform_test"
    password = "terraform"
    admin = false
}
`

var testAccUserConfig_grantUpdate = `
resource "influxdb_database" "green" {
    name = "terraform-green"
}

resource "influxdb_database" "blue" {
    name = "terraform-blue"
}

resource "influxdb_user" "test" {
    name = "terraform_test"
    password = "terraform"

    grant {
      database = "${influxdb_database.green.name}"
      privilege = "write"
    }

    grant {
      database = "${influxdb_database.blue.name}"
      privilege = "read"
    }
}
`
