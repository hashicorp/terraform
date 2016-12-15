package ns1

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/data"
)

func TestAccNS1DataFeed_Basic(t *testing.T) {
	var dataFeed data.Feed

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNS1DataFeedDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNS1DataFeed_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNS1DataFeedExists("ns1_datafeed.foobar", "ns1_datasource.api", &dataFeed),
					testAccCheckNS1DataFeedAttributes(&dataFeed),
					resource.TestCheckResourceAttr("ns1_datafeed.foobar", "name", "terraform test"),
					resource.TestCheckResourceAttr("ns1_datafeed.foobar", "config.label", "example"),
				),
			},
		},
	})
}

func TestAccNS1DataFeed_Updated(t *testing.T) {
	var dataFeed data.Feed

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNS1DataFeedDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNS1DataFeed_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNS1DataFeedExists("ns1_datafeed.foobar", "ns1_datasource.api", &dataFeed),
					testAccCheckNS1DataFeedAttributes(&dataFeed),
				),
			},
			resource.TestStep{
				Config: testAccNS1DataFeed_updated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNS1DataFeedExists("ns1_datafeed.foobar", "ns1_datasource.api", &dataFeed),
					testAccCheckNS1DataFeedAttributesUpdated(&dataFeed),
					resource.TestCheckResourceAttr("ns1_datafeed.foobar", "name", "terraform test updated"),
					resource.TestCheckResourceAttr("ns1_datafeed.foobar", "config.label", "example_updated"),
				),
			},
		},
	})
}

func testAccCheckNS1DataFeedExists(n string, dsrc string, dataFeed *data.Feed) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		ds, ok := s.RootModule().Resources[dsrc]
		if !ok {
			return fmt.Errorf("Not found: %s", dsrc)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("NoID is set")
		}

		if ds.Primary.ID == "" {
			return fmt.Errorf("NoID is set for the datasource")
		}

		client := testAccProvider.Meta().(*ns1.Client)

		foundFeed, _, err := client.DataFeeds.Get(ds.Primary.Attributes["id"], rs.Primary.Attributes["id"])
		if err != nil {
			return err
		}

		if foundFeed.Name != rs.Primary.Attributes["name"] {
			return fmt.Errorf("DataFeed not found")
		}

		*dataFeed = *foundFeed

		return nil
	}
}

func testAccCheckNS1DataFeedDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ns1.Client)

	var dataFeedID, dataSourceID string
	for _, rs := range s.RootModule().Resources {

		if rs.Type == "ns1_datasource" {
			dataSourceID = rs.Primary.Attributes["id"]
		}

		if rs.Type == "ns1_datafeed" {
			dataFeedID = rs.Primary.Attributes["id"]
		}
	}

	_, _, err := client.DataFeeds.Get(dataSourceID, dataFeedID)
	if err == nil {
		return fmt.Errorf("DataFeed still exists: %s", dataFeedID)
	}

	return nil
}

func testAccCheckNS1DataFeedAttributes(dataFeed *data.Feed) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if dataFeed.Name != "terraform test" {
			return fmt.Errorf("Bad value datafeed.Name: %s", dataFeed.Name)
		}
		if dataFeed.Config["label"] != "example" {
			return fmt.Errorf("Bad value datafeed.Config['label']: %s", dataFeed.Config["label"])
		}

		return nil
	}
}

func testAccCheckNS1DataFeedAttributesUpdated(dataFeed *data.Feed) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if dataFeed.Name != "terraform test updated" {
			return fmt.Errorf("Bad value datafeed.Name: %s", dataFeed.Name)
		}
		if dataFeed.Config["label"] != "example_updated" {
			return fmt.Errorf("Bad value datafeed.Config['label']: %s", dataFeed.Config["label"])
		}

		return nil
	}
}

const testAccNS1DataFeed_basic = `
resource "ns1_datasource" "api" {
  name = "terraform test"
  type = "nsone_v1"
}

resource "ns1_datafeed" "foobar" {
  name = "terraform test"
  source_id = "${ns1_datasource.api.id}"
  config {
    label = "example"
  }
}`

const testAccNS1DataFeed_updated = `
resource "ns1_datasource" "api" {
  name = "terraform test"
  type = "nsone_v1"
}

resource "ns1_datafeed" "foobar" {
  name = "terraform test updated"
  source_id = "${ns1_datasource.api.id}"
  config {
    label = "example_updated"
  }
}`
