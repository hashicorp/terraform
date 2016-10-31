package ns1

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	nsone "gopkg.in/ns1/ns1-go.v2/rest"
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
					testAccCheckDataFeedState("name", "terraform test"),
					testAccCheckDataFeedExists("ns1_datafeed.foobar", "ns1_datasource.api", &dataFeed),
					testAccCheckDataFeedAttributes(&dataFeed),
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
					testAccCheckDataFeedState("name", "terraform test"),
					testAccCheckDataFeedExists("ns1_datafeed.foobar", "ns1_datasource.api", &dataFeed),
					testAccCheckDataFeedAttributes(&dataFeed),
				),
			},
			resource.TestStep{
				Config: testAccDataFeedUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataFeedState("name", "terraform test"),
					testAccCheckDataFeedExists("ns1_datafeed.foobar", "ns1_datasource.api", &dataFeed),
					testAccCheckDataFeedAttributesUpdated(&dataFeed),
				),
			},
		},
	})
}

func testAccCheckDataFeedState(key, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["ns1_datafeed.foobar"]
		if !ok {
			return fmt.Errorf("Not found: %s", "ns1_datafeed.foobar")
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

		client := testAccProvider.Meta().(*nsone.Client)

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
	client := testAccProvider.Meta().(*nsone.Client)

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

func testAccCheckDataFeedAttributes(dataFeed *data.Feed) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if dataFeed.Config["label"] != "exampledc2" {
			return fmt.Errorf("Bad value : %s", dataFeed.Config["label"])
		}

		return nil
	}
}

func testAccCheckDataFeedAttributesUpdated(dataFeed *data.Feed) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if dataFeed.Config["label"] != "exampledc3" {
			return fmt.Errorf("Bad value : %s", dataFeed.Config["label"])
		}

		return nil
	}
}

const testAccDataFeedBasic = `
resource "ns1_datasource" "api" {
	name = "terraform test"
	sourcetype = "ns1_v1"
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
	sourcetype = "ns1_v1"
}

resource "ns1_datafeed" "foobar" {
	name = "terraform test"
	source_id = "${ns1_datasource.api.id}"
  config {
		label = "exampledc3"
	}
}`
