package mysql

import (
	"fmt"
	"log"
	"strings"
	"testing"

	mysqlc "github.com/ziutek/mymysql/mysql"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccGrant(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGrantCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccGrantConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccPrivilegeExists("mysql_grant.test", "SELECT"),
					resource.TestCheckResourceAttr("mysql_grant.test", "user", "jdoe"),
					resource.TestCheckResourceAttr("mysql_grant.test", "host", "example.com"),
					resource.TestCheckResourceAttr("mysql_grant.test", "database", "foo"),
				),
			},
		},
	})
}

func testAccPrivilegeExists(rn string, privilege string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("grant id not set")
		}

		id := strings.Split(rs.Primary.ID, ":")
		userhost := strings.Split(id[0], "@")
		user := userhost[0]
		host := userhost[1]

		conn := testAccProvider.Meta().(*providerConfiguration).Conn
		stmtSQL := fmt.Sprintf("SHOW GRANTS for '%s'@'%s'", user, host)
		log.Println("Executing statement:", stmtSQL)
		rows, _, err := conn.Query(stmtSQL)
		if err != nil {
			return fmt.Errorf("error reading grant: %s", err)
		}

		if len(rows) == 0 {
			return fmt.Errorf("grant not found for '%s'@'%s'", user, host)
		}

		privilegeFound := false
		for _, row := range rows {
			log.Printf("Result Row: %s", row[0])
			privIndex := strings.Index(string(row[0].([]byte)), privilege)
			if privIndex != -1 {
				privilegeFound = true
			}
		}

		if !privilegeFound {
			return fmt.Errorf("grant no found for '%s'@'%s'", user, host)
		}

		return nil
	}
}

func testAccGrantCheckDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*providerConfiguration).Conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "mysql_grant" {
			continue
		}

		id := strings.Split(rs.Primary.ID, ":")
		userhost := strings.Split(id[0], "@")
		user := userhost[0]
		host := userhost[1]

		stmtSQL := fmt.Sprintf("SHOW GRANTS for '%s'@'%s'", user, host)
		log.Println("Executing statement:", stmtSQL)
		rows, _, err := conn.Query(stmtSQL)
		if err != nil {
			if mysqlErr, ok := err.(*mysqlc.Error); ok {
				if mysqlErr.Code == mysqlc.ER_NONEXISTING_GRANT {
					return nil
				}
			}

			return fmt.Errorf("error reading grant: %s", err)
		}

		if len(rows) != 0 {
			return fmt.Errorf("grant still exists for'%s'@'%s'", user, host)
		}
	}
	return nil
}

const testAccGrantConfig_basic = `
resource "mysql_user" "test" {
        user = "jdoe"
				host = "example.com"
				password = "password"
}

resource "mysql_grant" "test" {
        user = "${mysql_user.test.user}"
        host = "${mysql_user.test.host}"
        database = "foo"
        privileges = ["UPDATE", "SELECT"]
}
`
