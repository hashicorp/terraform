package influxdb

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDatabase(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDatabaseConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"influxdb_database.test", "name", "terraform-test",
					),
				),
			},
		},
	})
}

var testAccDatabaseConfig = `

resource "influxdb_database" "test" {
    name = "terraform-test"
}

`
