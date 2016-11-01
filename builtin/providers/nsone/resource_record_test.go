package nsone

import (
	"fmt"
	"reflect"
	"sort"
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
					testAccCheckRecordState("domain", "test.terraform-record-test.io"),
					testAccCheckRecordExists("nsone_record.it", &record),
					testAccCheckRecordTTL(&record, 60),
					testAccCheckRecordRegionName(&record, []string{"cal"}),
					testAccCheckRecordAnswerMetaWeight(&record, 10),
					testAccCheckRecordAnswerRdata(&record, "test1.terraform-record-test.io"),
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
					testAccCheckRecordState("domain", "test.terraform-record-test.io"),
					testAccCheckRecordExists("nsone_record.it", &record),
					testAccCheckRecordTTL(&record, 60),
					testAccCheckRecordRegionName(&record, []string{"cal"}),
					testAccCheckRecordAnswerMetaWeight(&record, 10),
					testAccCheckRecordAnswerRdata(&record, "test1.terraform-record-test.io"),
				),
			},
			resource.TestStep{
				Config: testAccRecordUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRecordState("domain", "test.terraform-record-test.io"),
					testAccCheckRecordExists("nsone_record.it", &record),
					testAccCheckRecordTTL(&record, 120),
					testAccCheckRecordRegionName(&record, []string{"ny", "wa"}),
					testAccCheckRecordAnswerMetaWeight(&record, 5),
					testAccCheckRecordAnswerRdata(&record, "test2.terraform-record-test.io"),
				),
			},
		},
	})
}

func testAccCheckRecordState(key, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["nsone_record.it"]
		if !ok {
			return fmt.Errorf("Not found: nsone_record.it")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p := rs.Primary
		if p.Attributes[key] != value {
			return fmt.Errorf(
				"%v: want: %v got: %v", key, value, p.Attributes[key])
		}

		return nil
	}
}

func testAccCheckRecordExists(n string, record *nsone.Record) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %v", n)
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
		return fmt.Errorf("Record still exists: %#v", foundRecord)
	}

	return nil
}

func testAccCheckRecordTTL(r *nsone.Record, expected int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if r.Ttl != expected {
			return fmt.Errorf("TTL: got: %#v want: %#v", r.Ttl, expected)
		}
		return nil
	}
}

func testAccCheckRecordRegionName(r *nsone.Record, expected []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		regions := make([]string, len(r.Regions))

		i := 0
		for k := range r.Regions {
			regions[i] = k
			i++
		}
		sort.Strings(regions)
		sort.Strings(expected)
		if !reflect.DeepEqual(regions, expected) {
			return fmt.Errorf("Regions: got: %#v want: %#v", regions, expected)
		}
		return nil
	}
}

func testAccCheckRecordAnswerMetaWeight(r *nsone.Record, expected float64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		recordAnswer := r.Answers[0]
		recordMetas := recordAnswer.Meta
		weight := recordMetas["weight"].(float64)
		if weight != expected {
			return fmt.Errorf("Answers[0].Meta.Weight: got: %#v want: %#v", weight, expected)
		}
		return nil
	}
}

func testAccCheckRecordAnswerRdata(r *nsone.Record, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		recordAnswer := r.Answers[0]
		recordAnswerString := recordAnswer.Answer[0]
		if recordAnswerString != expected {
			return fmt.Errorf("Answers[0].Rdata[0]: got: %#v want: %#v", recordAnswerString, expected)
		}
		return nil
	}
}

const testAccRecordBasic = `
resource "nsone_record" "it" {
  zone              = "${nsone_zone.test.zone}"
  domain            = "test.terraform-record-test.io"
  type              = "CNAME"
  ttl               = 60

  answers {
    answer = "test1.terraform-record-test.io"
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

  regions {
    name     = "cal"
    us_state = "CA"
  }

  filters {
    filter = "up"
  }

  filters {
    filter = "geotarget_country"
  }
}

resource "nsone_zone" "test" {
  zone = "terraform-record-test.io"
}
`

const testAccRecordUpdated = `
resource "nsone_record" "it" {
  zone              = "${nsone_zone.test.zone}"
  domain            = "test.terraform-record-test.io"
  type              = "CNAME"
  ttl               = 120
  use_client_subnet = true

  answers {
    answer = "test2.terraform-record-test.io"
    region = "ny"

    meta {
      field = "weight"
      value = "5"
    }

    meta {
      field = "up"
      value = "1"
    }
  }

  regions {
    name     = "wa"
    us_state = "WA"
  }

  regions {
    name     = "ny"
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
  zone = "terraform-record-test.io"
}
`
