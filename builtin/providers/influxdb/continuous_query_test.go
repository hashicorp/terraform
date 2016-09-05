package influxdb

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/influxdata/influxdb/client"
)

func TestAccInfluxDBContiuousQuery(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContiuousQueryConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContiuousQueryExists("influxdb_continuous_query.minnie"),
					resource.TestCheckResourceAttr(
						"influxdb_continuous_query.minnie", "name", "minnie",
					),
					resource.TestCheckResourceAttr(
						"influxdb_continuous_query.minnie", "database", "terraform-test",
					),
					resource.TestCheckResourceAttr(
						"influxdb_continuous_query.minnie", "query", "SELECT min(mouse) INTO min_mouse FROM zoo GROUP BY time(30m)",
					),
				),
			},
		},
	})
}

func testAccCheckContiuousQueryExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ContiuousQuery id set")
		}

		conn := testAccProvider.Meta().(*client.Client)

		query := client.Query{
			Command: "SHOW CONTINUOUS QUERIES",
		}

		resp, err := conn.Query(query)
		if err != nil {
			return err
		}

		if resp.Err != nil {
			return resp.Err
		}

		for _, series := range resp.Results[0].Series {
			if series.Name == rs.Primary.Attributes["database"] {
				for _, result := range series.Values {
					if result[0].(string) == rs.Primary.Attributes["name"] {
						return nil
					}
				}
			}
		}

		return fmt.Errorf("ContiuousQuery %q does not exist", rs.Primary.Attributes["name"])
	}
}

var testAccContiuousQueryConfig = `

resource "influxdb_database" "test" {
    name = "terraform-test"
}

resource "influxdb_continuous_query" "minnie" {
    name = "minnie"
    database = "${influxdb_database.test.name}"
    query = "SELECT min(mouse) INTO min_mouse FROM zoo GROUP BY time(30m)"
}

`
