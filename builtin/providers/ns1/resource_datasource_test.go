package ns1

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/data"
)

func TestAccDataSource_basic(t *testing.T) {
	var dataSource data.Source
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDataSourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataSourceExists("ns1_datasource.foobar", &dataSource),
					testAccCheckDataSourceName(&dataSource, "terraform test"),
					testAccCheckDataSourceType(&dataSource, "nsone_v1"),
				),
			},
		},
	})
}

func TestAccDataSource_updated(t *testing.T) {
	var dataSource data.Source
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDataSourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataSourceExists("ns1_datasource.foobar", &dataSource),
					testAccCheckDataSourceName(&dataSource, "terraform test"),
					testAccCheckDataSourceType(&dataSource, "nsone_v1"),
				),
			},
			resource.TestStep{
				Config: testAccDataSourceUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataSourceExists("ns1_datasource.foobar", &dataSource),
					testAccCheckDataSourceName(&dataSource, "terraform test"),
					testAccCheckDataSourceType(&dataSource, "nsone_monitoring"),
				),
			},
		},
	})
}

func testAccCheckDataSourceExists(n string, dataSource *data.Source) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("NoID is set")
		}

		client := testAccProvider.Meta().(*ns1.Client)

		foundSource, _, err := client.DataSources.Get(rs.Primary.Attributes["id"])

		p := rs.Primary

		if err != nil {
			return err
		}

		if foundSource.Name != p.Attributes["name"] {
			return fmt.Errorf("Datasource not found")
		}

		*dataSource = *foundSource

		return nil
	}
}

func testAccCheckDataSourceDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ns1.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ns1_datasource" {
			continue
		}

		_, _, err := client.DataSources.Get(rs.Primary.Attributes["id"])

		if err == nil {
			return fmt.Errorf("Datasource still exists")
		}
	}

	return nil
}

func testAccCheckDataSourceName(dataSource *data.Source, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if dataSource.Name != expected {
			return fmt.Errorf("Name: got: %#v want: %#v", dataSource.Name, expected)
		}

		return nil
	}
}

func testAccCheckDataSourceType(dataSource *data.Source, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if dataSource.Type != expected {
			return fmt.Errorf("Type: got: %#v want: %#v", dataSource.Type, expected)
		}

		return nil
	}
}

const testAccDataSourceBasic = `
resource "ns1_datasource" "foobar" {
	name = "terraform test"
	sourcetype = "nsone_v1"
}`

const testAccDataSourceUpdated = `
resource "ns1_datasource" "foobar" {
	name = "terraform test"
	sourcetype = "nsone_monitoring"
}`
