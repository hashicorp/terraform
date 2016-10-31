package ns1

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	nsone "gopkg.in/ns1/ns1-go.v2/rest"
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
					testAccCheckDataSourceState("name", "terraform test"),
					testAccCheckDataSourceExists("ns1_datasource.foobar", &dataSource),
					testAccCheckDataSourceAttributes(&dataSource),
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
					testAccCheckDataSourceState("name", "terraform test"),
					testAccCheckDataSourceExists("ns1_datasource.foobar", &dataSource),
					testAccCheckDataSourceAttributes(&dataSource),
				),
			},
			resource.TestStep{
				Config: testAccDataSourceUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataSourceState("name", "terraform test"),
					testAccCheckDataSourceExists("ns1_datasource.foobar", &dataSource),
					testAccCheckDataSourceAttributesUpdated(&dataSource),
				),
			},
		},
	})
}

func testAccCheckDataSourceState(key, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["ns1_datasource.foobar"]
		if !ok {
			return fmt.Errorf("Not found: %s", "ns1_zone.foobar")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p := rs.Primary
		if p.Attributes[key] != value {
			return fmt.Errorf(
				"%s != %s (actual: %s)", key, value, p.Attributes[key])
		}

		return nil
	}
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

		client := testAccProvider.Meta().(*nsone.Client)

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
	client := testAccProvider.Meta().(*nsone.Client)

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

func testAccCheckDataSourceAttributes(dataSource *data.Source) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if dataSource.Type != "ns1_v1" {
			return fmt.Errorf("Bad value : %s", dataSource.Type)
		}

		return nil
	}
}

func testAccCheckDataSourceAttributesUpdated(dataSource *data.Source) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if dataSource.Type != "ns1_monitoring" {
			return fmt.Errorf("Bad value : %s", dataSource.Type)
		}

		return nil
	}
}

const testAccDataSourceBasic = `
resource "ns1_datasource" "foobar" {
	name = "terraform test"
	sourcetype = "ns1_v1"
}`

const testAccDataSourceUpdated = `
resource "ns1_datasource" "foobar" {
	name = "terraform test"
	sourcetype = "ns1_monitoring"
}`
