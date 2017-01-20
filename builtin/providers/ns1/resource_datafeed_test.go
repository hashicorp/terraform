package ns1

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/data"
)

func TestAccDataFeed_basic(t *testing.T) {
	var dataFeed data.Feed
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDataFeedDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataFeedBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataFeedExists("ns1_datafeed.foobar", "ns1_datasource.api", &dataFeed),
					testAccCheckDataFeedName(&dataFeed, "terraform test"),
					testAccCheckDataFeedConfig(&dataFeed, "label", "exampledc2"),
				),
			},
		},
	})
}

func TestAccDataFeed_updated(t *testing.T) {
	var dataFeed data.Feed
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDataFeedDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataFeedBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataFeedExists("ns1_datafeed.foobar", "ns1_datasource.api", &dataFeed),
					testAccCheckDataFeedName(&dataFeed, "terraform test"),
					testAccCheckDataFeedConfig(&dataFeed, "label", "exampledc2"),
				),
			},
			resource.TestStep{
				Config: testAccDataFeedUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataFeedExists("ns1_datafeed.foobar", "ns1_datasource.api", &dataFeed),
					testAccCheckDataFeedName(&dataFeed, "terraform test"),
					testAccCheckDataFeedConfig(&dataFeed, "label", "exampledc3"),
				),
			},
		},
	})
}

func testAccCheckDataFeedExists(n string, dsrc string, dataFeed *data.Feed) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		ds, ok := s.RootModule().Resources[dsrc]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("NoID is set")
		}

		if ds.Primary.ID == "" {
			return fmt.Errorf("NoID is set for the datasource")
		}

		client := testAccProvider.Meta().(*ns1.Client)

		foundFeed, _, err := client.DataFeeds.Get(ds.Primary.Attributes["id"], rs.Primary.Attributes["id"])

		p := rs.Primary

		if err != nil {
			return err
		}

		if foundFeed.Name != p.Attributes["name"] {
			return fmt.Errorf("DataFeed not found")
		}

		*dataFeed = *foundFeed

		return nil
	}
}

func testAccCheckDataFeedDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ns1.Client)

	var dataFeedID string
	var dataSourceID string

	for _, rs := range s.RootModule().Resources {

		if rs.Type == "ns1_datasource" {
			dataSourceID = rs.Primary.Attributes["id"]
		}

		if rs.Type == "ns1_datafeed" {
			dataFeedID = rs.Primary.Attributes["id"]
		}
	}

	df, _, _ := client.DataFeeds.Get(dataSourceID, dataFeedID)

	if df != nil {
		return fmt.Errorf("DataFeed still exists: %#v", df)
	}

	return nil
}

func testAccCheckDataFeedName(dataFeed *data.Feed, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if dataFeed.Name != expected {
			return fmt.Errorf("Name: got: %#v want: %#v", dataFeed.Name, expected)
		}

		return nil
	}
}

func testAccCheckDataFeedConfig(dataFeed *data.Feed, key, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if dataFeed.Config[key] != expected {
			return fmt.Errorf("Config[%s]: got: %#v, want: %s", key, dataFeed.Config[key], expected)
		}

		return nil
	}
}

const testAccDataFeedBasic = `
resource "ns1_datasource" "api" {
  name = "terraform test"
  sourcetype = "nsone_v1"
}

resource "ns1_datafeed" "foobar" {
  name = "terraform test"
  source_id = "${ns1_datasource.api.id}"
  config {
    label = "exampledc2"
  }
}`

const testAccDataFeedUpdated = `
resource "ns1_datasource" "api" {
  name = "terraform test"
  sourcetype = "nsone_v1"
}

resource "ns1_datafeed" "foobar" {
  name = "terraform test"
  source_id = "${ns1_datasource.api.id}"
  config {
    label = "exampledc3"
  }
}`
