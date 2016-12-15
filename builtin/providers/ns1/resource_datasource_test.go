package ns1

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/data"
)

func TestAccNS1DataSource_Basic(t *testing.T) {
	var dataSource data.Source

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNS1DataSourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNS1DataSource_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNS1DataSourceExists("ns1_datasource.foobar", &dataSource),
					testAccCheckNS1DataSourceAttributes(&dataSource),
				),
			},
		},
	})
}

func TestAccNS1DataSource_Updated(t *testing.T) {
	var dataSource data.Source

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNS1DataSourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNS1DataSource_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNS1DataSourceExists("ns1_datasource.foobar", &dataSource),
					testAccCheckNS1DataSourceAttributes(&dataSource),
				),
			},
			resource.TestStep{
				Config: testAccNS1DataSource_updated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNS1DataSourceExists("ns1_datasource.foobar", &dataSource),
					testAccCheckNS1DataSourceAttributesUpdated(&dataSource),
				),
			},
		},
	})
}

func testAccCheckNS1DataSourceExists(n string, dataSource *data.Source) resource.TestCheckFunc {
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
		if err != nil {
			return err
		}

		if foundSource.Name != rs.Primary.Attributes["name"] {
			return fmt.Errorf("Datasource not found")
		}

		*dataSource = *foundSource

		return nil
	}
}

func testAccCheckNS1DataSourceDestroy(s *terraform.State) error {
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

func testAccCheckNS1DataSourceAttributes(dataSource *data.Source) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if dataSource.Name != "terraform test" {
			return fmt.Errorf("Bad value datasource.Name: %s", dataSource.Name)
		}
		if dataSource.Type != "nsone_v1" {
			return fmt.Errorf("Bad value datasource.Type: %s", dataSource.Type)
		}

		return nil
	}
}

func testAccCheckNS1DataSourceAttributesUpdated(dataSource *data.Source) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if dataSource.Name != "terraform test" {
			return fmt.Errorf("Bad value datasource.Name: %s", dataSource.Name)
		}
		if dataSource.Type != "nsone_monitoring" {
			return fmt.Errorf("Bad value datasource.Type: %s", dataSource.Type)
		}

		return nil
	}
}

const testAccNS1DataSource_basic = `
resource "ns1_datasource" "foobar" {
  name = "terraform test"
  type = "nsone_v1"
}`

const testAccNS1DataSource_updated = `
resource "ns1_datasource" "foobar" {
  name = "terraform test"
  type = "nsone_monitoring"
}`
