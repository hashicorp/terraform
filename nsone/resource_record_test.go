package nsone

import (
	"fmt"
	"testing"

	"github.com/bobtfish/go-nsone-api"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccRecord_basic(t *testing.T) {
	var record nsone.Record
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRecord_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRecordState("name", "terraform test"),
					testAccCheckRecordExists("nsone_record.foobar", "nsone_zone.test", &record),
					testAccCheckRecordAttributes(&record),
				),
			},
		},
	})
}

func TestAccRecord_updated(t *testing.T) {
	var record nsone.Record
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRecord_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRecordState("name", "terraform test"),
					testAccCheckRecordExists("nsone_record.foobar", "nsone_zone.test", &record),
					testAccCheckRecordAttributes(&record),
				),
			},
			resource.TestStep{
				Config: testAccRecord_updated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRecordState("name", "terraform test"),
					testAccCheckRecordExists("nsone_record.foobar", "nsone_zone.test", &record),
					testAccCheckRecordAttributesUpdated(&record),
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

	var dataFeedId string
	var dataSourceId string

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "nsone_datasource" || rs.Type != "nsone_datafeed" {
			continue
		}

		if rs.Type == "nsone_datasource" {
			dataSourceId = rs.Primary.Attributes["id"]
		}

		if rs.Type == "nsone_datafeed" {
			dataFeedId = rs.Primary.Attributes["id"]
		}
	}

	df, _ := client.GetDataFeed(dataSourceId, dataFeedId)

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

const testAccRecord_basic = `
resource "nsone_record" "foobar" {
    zone = "terraform.io"
		domain = "test.terraform.io"
	  type = "CNAME"
		ttl = 60
		use_client_subnet = false
    answers {
      answer = "test1.terraform.io"
      region = "cal"
      meta {
        field = "weight"
        value = "10"
      }
      meta {
        field = "up"
        value = "1"
      }
    }
    answers {
      answer = "test2.terraform.io"
      region = "ny"
      meta {
        field = "weight"
        value = "10"
      }
      meta {
        field = "up"
        value = "1"
      }
    }
    regions {
      name = "cal"
      us_state = "CA"
    }
    regions {
      name = "ny"
      us_state = "NY"
    }

    filters {
        filter = "up"
    }
    filters {
        filter = "geotarget_country"
    }
}
resource "nsone_zone" "test" {
  zone = "terraform.io"
}`

const testAccRecord_updated = `
resource "nsone_record" "foobar" {
	zone = "terraform.io"
	domain = "test.terraform.io"
	type = "CNAME"
	ttl = 60
	use_client_subnet = false
	answers {
		answer = "test1.terraform.io"
		region = "cal"
		meta {
			field = "weight"
			value = "10"
		}
		meta {
			field = "up"
			value = "1"
		}
	}
	answers {
		answer = "test2.terraform.io"
		region = "ny"
		meta {
			field = "weight"
			value = "10"
		}
		meta {
			field = "up"
			value = "1"
		}
	}
	regions {
		name = "cal"
		us_state = "CA"
	}
	regions {
		name = "ny"
		us_state = "NY"
	}

	filters {
		filter = "up"
	}
	filters {
		filter = "geotarget_country"
	}
}
resource "nsone_zone" "test" {
	zone = "mycompany.co.uk"
}`
