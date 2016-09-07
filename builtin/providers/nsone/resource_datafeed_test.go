package nsone

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	nsone "gopkg.in/ns1/ns1-go.v2/rest"
)

func TestAccDataFeed_basic(t *testing.T) {
	var dataFeed nsone.DataFeed
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDataFeedDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataFeedBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataFeedState("name", "terraform test"),
					testAccCheckDataFeedExists("nsone_datafeed.foobar", "nsone_datasource.api", &dataFeed),
					testAccCheckDataFeedAttributes(&dataFeed),
				),
			},
		},
	})
}

func TestAccDataFeed_updated(t *testing.T) {
	var dataFeed nsone.DataFeed
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDataFeedDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataFeedBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataFeedState("name", "terraform test"),
					testAccCheckDataFeedExists("nsone_datafeed.foobar", "nsone_datasource.api", &dataFeed),
					testAccCheckDataFeedAttributes(&dataFeed),
				),
			},
			resource.TestStep{
				Config: testAccDataFeedUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataFeedState("name", "terraform test"),
					testAccCheckDataFeedExists("nsone_datafeed.foobar", "nsone_datasource.api", &dataFeed),
					testAccCheckDataFeedAttributesUpdated(&dataFeed),
				),
			},
		},
	})
}

func testAccCheckDataFeedState(key, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["nsone_datafeed.foobar"]
		if !ok {
			return fmt.Errorf("Not found: %s", "nsone_datafeed.foobar")
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

func testAccCheckDataFeedExists(n string, dsrc string, dataFeed *nsone.DataFeed) resource.TestCheckFunc {
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

		client := testAccProvider.Meta().(*nsone.APIClient)

		foundFeed, err := client.GetDataFeed(ds.Primary.Attributes["id"], rs.Primary.Attributes["id"])

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
	client := testAccProvider.Meta().(*nsone.APIClient)

	var dataFeedID string
	var dataSourceID string

	for _, rs := range s.RootModule().Resources {

		if rs.Type == "nsone_datasource" {
			dataSourceID = rs.Primary.Attributes["id"]
		}

		if rs.Type == "nsone_datafeed" {
			dataFeedID = rs.Primary.Attributes["id"]
		}
	}

	df, _ := client.GetDataFeed(dataSourceID, dataFeedID)

	if df.Id != "" {
		return fmt.Errorf("DataFeed still exists")
	}

	return nil
}

func testAccCheckDataFeedAttributes(dataFeed *nsone.DataFeed) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if dataFeed.Config["label"] != "exampledc2" {
			return fmt.Errorf("Bad value : %s", dataFeed.Config["label"])
		}

		return nil
	}
}

func testAccCheckDataFeedAttributesUpdated(dataFeed *nsone.DataFeed) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if dataFeed.Config["label"] != "exampledc3" {
			return fmt.Errorf("Bad value : %s", dataFeed.Config["label"])
		}

		return nil
	}
}

const testAccDataFeedBasic = `
resource "nsone_datasource" "api" {
	name = "terraform test"
	sourcetype = "nsone_v1"
}

resource "nsone_datafeed" "foobar" {
	name = "terraform test"
	source_id = "${nsone_datasource.api.id}"
	config {
		label = "exampledc2"
	}
}`

const testAccDataFeedUpdated = `
resource "nsone_datasource" "api" {
	name = "terraform test"
	sourcetype = "nsone_v1"
}

resource "nsone_datafeed" "foobar" {
	name = "terraform test"
	source_id = "${nsone_datasource.api.id}"
  config {
		label = "exampledc3"
	}
}`
