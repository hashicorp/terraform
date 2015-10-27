package postgresql

import (
  "fmt"
  "testing"
  "database/sql"

  "github.com/hashicorp/terraform/helper/resource"
  "github.com/hashicorp/terraform/terraform"
)

func TestAccPostgresqlDb_Basic(t *testing.T) {

  resource.Test(t, resource.TestCase{
    PreCheck:     func() { testAccPreCheck(t) },
    Providers:    testAccProviders,
    CheckDestroy: testAccCheckPostgresqlDbDestroy,
    Steps: []resource.TestStep{
      resource.TestStep{
        Config: testAccPostgresqlDbConfig,
        Check: resource.ComposeTestCheckFunc(
          testAccCheckPostgresqlDbExists("postgresql_db.mydb", "myrole"),
          resource.TestCheckResourceAttr(
            "postgresql_db.mydb", "name", "mydb"),
          resource.TestCheckResourceAttr(
            "postgresql_db.mydb", "owner", "myrole"),
        ),
      },
    },
  })
}

func testAccCheckPostgresqlDbDestroy(s *terraform.State) error {
  client := testAccProvider.Meta().(*sql.DB)

  for _, rs := range s.RootModule().Resources {
    if rs.Type != "postgresql_db" {
      continue
    }

    exists, err := checkDbExists(client, rs.Primary.ID)

    if err != nil {
      return fmt.Errorf("Error checking db %s", err)
    }

    if exists {
      return fmt.Errorf("Db still exists after destroy")
    }
  }

  return nil
}

func testAccCheckPostgresqlDbExists(n string, owner string) resource.TestCheckFunc {
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

    client := testAccProvider.Meta().(*sql.DB)
    exists, err := checkDbExists(client, rs.Primary.ID)

    if err != nil {
      return fmt.Errorf("Error checking db %s", err)
    }

    if !exists {
      return fmt.Errorf("Db not found")
    }

    return nil
  }
}

func checkDbExists(conn *sql.DB, dbName string) (bool, error) {
    var _rez int
    err := conn.QueryRow("SELECT 1 from pg_database d WHERE datname=$1", dbName).Scan(&_rez)
    switch {
    case err == sql.ErrNoRows:
      return false, nil
    case err != nil:
      return false, fmt.Errorf("Error reading info about database: %s", err)
    default:
      return true, nil
    }
}

var testAccPostgresqlDbConfig = `
resource "postgresql_role" "myrole" {
  name = "myrole"
  login = true
}

resource "postgresql_db" "mydb" {
   name = "mydb"
   owner = "${postgresql_role.myrole.name}"
}
`