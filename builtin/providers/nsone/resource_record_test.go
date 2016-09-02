package nsone

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	nsone "gopkg.in/ns1/ns1-go.v2/rest"
)

func TestAccRecord_basic(t *testing.T) {
	var record nsone.Record
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRecordBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRecordState("domain", "test.terraform.io"),
					testAccCheckRecordExists("nsone_record.foobar", &record),
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
				Config: testAccRecordBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRecordState("domain", "test.terraform.io"),
					testAccCheckRecordExists("nsone_record.foobar", &record),
					testAccCheckRecordAttributes(&record),
				),
			},
			resource.TestStep{
				Config: testAccRecordUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRecordState("domain", "test.terraform.io"),
					testAccCheckRecordExists("nsone_record.foobar", &record),
					testAccCheckRecordAttributesUpdated(&record),
				),
			},
		},
	})
}

func testAccCheckRecordState(key, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["nsone_record.foobar"]
		if !ok {
			return fmt.Errorf("Not found: %s", "nsone_record.foobar")
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

func testAccCheckRecordExists(n string, record *nsone.Record) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("NoID is set")
		}

		client := testAccProvider.Meta().(*nsone.APIClient)

		p := rs.Primary

		foundRecord, err := client.GetRecord(p.Attributes["zone"], p.Attributes["domain"], p.Attributes["type"])

		if err != nil {
			// return err
			return fmt.Errorf("Record not found")
		}

		if foundRecord.Domain != p.Attributes["domain"] {
			return fmt.Errorf("Record not found")
		}

		*record = *foundRecord

		return nil
	}
}

func testAccCheckRecordDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*nsone.APIClient)

	var recordDomain string
	var recordZone string
	var recordType string

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "nsone_record" {
			continue
		}

		if rs.Type == "nsone_record" {
			recordType = rs.Primary.Attributes["type"]
			recordDomain = rs.Primary.Attributes["domain"]
			recordZone = rs.Primary.Attributes["zone"]
		}
	}

	foundRecord, _ := client.GetRecord(recordDomain, recordZone, recordType)

	if foundRecord.Id != "" {
		return fmt.Errorf("Record still exists")
	}

	return nil
}

func testAccCheckRecordAttributes(record *nsone.Record) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if record.Ttl != 60 {
			return fmt.Errorf("Bad value : %d", record.Ttl)
		}

		recordAnswer := record.Answers[0]
		recordAnswerString := recordAnswer.Answer[0]

		if recordAnswerString != "test1.terraform.io" {
			return fmt.Errorf("Bad value : %s", record.Ttl)
		}

		if recordAnswer.Region != "cal" {
			return fmt.Errorf("Bad value : %s", recordAnswer.Region)
		}

		recordMetas := recordAnswer.Meta

		if recordMetas["weight"].(float64) != 10 {
			return fmt.Errorf("Bad value : %b", recordMetas["weight"].(float64))
		}

		return nil
	}
}

func testAccCheckRecordAttributesUpdated(record *nsone.Record) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if record.Ttl != 120 {
			return fmt.Errorf("Bad value : %s", record.Ttl)
		}

		recordAnswer := record.Answers[1]
		recordAnswerString := recordAnswer.Answer[0]

		if recordAnswerString != "test3.terraform.io" {
			return fmt.Errorf("Bad value for updated record: %s", recordAnswerString)
		}

		if recordAnswer.Region != "wa" {
			return fmt.Errorf("Bad value : %s", recordAnswer.Region)
		}

		recordMetas := recordAnswer.Meta

		if recordMetas["weight"].(float64) != 5 {
			return fmt.Errorf("Bad value : %b", recordMetas["weight"].(float64))
		}

		return nil
	}
}

const testAccRecordBasic = `
resource "nsone_record" "foobar" {
    zone = "${nsone_zone.test.zone}"
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

const testAccRecordUpdated = `
resource "nsone_record" "foobar" {
	zone = "terraform.io"
	domain = "test.terraform.io"
	type = "CNAME"
	ttl = 120
	use_client_subnet = false
	answers {
		answer = "test3.terraform.io"
		region = "wa"
		meta {
			field = "weight"
			value = "5"
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
		name = "wa"
		us_state = "WA"
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
