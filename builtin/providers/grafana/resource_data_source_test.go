package grafana

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	gapi "github.com/apparentlymart/go-grafana-api"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSource_basic(t *testing.T) {
	var dataSource gapi.DataSource

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDataSourceCheckDestroy(&dataSource),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceCheckExists("grafana_data_source.test", &dataSource),
					resource.TestCheckResourceAttr(
						"grafana_data_source.test", "type", "influxdb",
					),
					resource.TestMatchResourceAttr(
						"grafana_data_source.test", "id", regexp.MustCompile(`\d+`),
					),
				),
			},
		},
	})
}

func testAccDataSourceCheckExists(rn string, dataSource *gapi.DataSource) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("resource id is malformed")
		}

		client := testAccProvider.Meta().(*gapi.Client)
		gotDataSource, err := client.DataSource(id)
		if err != nil {
			return fmt.Errorf("error getting data source: %s", err)
		}

		*dataSource = *gotDataSource

		return nil
	}
}

func testAccDataSourceCheckDestroy(dataSource *gapi.DataSource) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*gapi.Client)
		_, err := client.DataSource(dataSource.Id)
		if err == nil {
			return fmt.Errorf("data source still exists")
		}
		return nil
	}
}

const testAccDataSourceConfig_basic = `
resource "grafana_data_source" "test" {
    type = "influxdb"
    name = "terraform-acc-test"
    database_name = "terraform-acc-test"
    url = "http://terraform-acc-test.invalid/"
    username = "terraform_user"
    password = "terraform_password"
}
`
