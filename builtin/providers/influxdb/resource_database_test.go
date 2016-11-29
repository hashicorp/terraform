package influxdb

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/influxdata/influxdb/client"
)

func TestAccInfluxDBDatabase(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDatabaseConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDatabaseExists("influxdb_database.test"),
					resource.TestCheckResourceAttr(
						"influxdb_database.test", "name", "terraform-test",
					),
				),
			},
		},
	})
}

func testAccCheckDatabaseExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No database id set")
		}

		conn := testAccProvider.Meta().(*client.Client)

		query := client.Query{
			Command: "SHOW DATABASES",
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

		return fmt.Errorf("Database %q does not exist", rs.Primary.Attributes["name"])
	}
}

var testAccDatabaseConfig = `

resource "influxdb_database" "test" {
    name = "terraform-test"
}

`
