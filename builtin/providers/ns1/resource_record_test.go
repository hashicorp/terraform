package ns1

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/dns"
)

func TestAccRecord_basic(t *testing.T) {
	var record dns.Record
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRecordBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRecordExists("ns1_record.it", &record),
					testAccCheckRecordDomain(&record, "test.terraform-record-test.io"),
					testAccCheckRecordTTL(&record, 60),
					testAccCheckRecordUseClientSubnet(&record, true),
					testAccCheckRecordRegionName(&record, []string{"cal"}),
					// testAccCheckRecordAnswerMetaWeight(&record, 10),
					testAccCheckRecordAnswerRdata(&record, "test1.terraform-record-test.io"),
				),
			},
		},
	})
}

func TestAccRecord_updated(t *testing.T) {
	var record dns.Record
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRecordBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRecordExists("ns1_record.it", &record),
					testAccCheckRecordDomain(&record, "test.terraform-record-test.io"),
					testAccCheckRecordTTL(&record, 60),
					testAccCheckRecordUseClientSubnet(&record, true),
					testAccCheckRecordRegionName(&record, []string{"cal"}),
					// testAccCheckRecordAnswerMetaWeight(&record, 10),
					testAccCheckRecordAnswerRdata(&record, "test1.terraform-record-test.io"),
				),
			},
			resource.TestStep{
				Config: testAccRecordUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRecordExists("ns1_record.it", &record),
					testAccCheckRecordDomain(&record, "test.terraform-record-test.io"),
					testAccCheckRecordTTL(&record, 120),
					testAccCheckRecordUseClientSubnet(&record, false),
					testAccCheckRecordRegionName(&record, []string{"ny", "wa"}),
					// testAccCheckRecordAnswerMetaWeight(&record, 5),
					testAccCheckRecordAnswerRdata(&record, "test2.terraform-record-test.io"),
				),
			},
		},
	})
}

func testAccCheckRecordExists(n string, record *dns.Record) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %v", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("NoID is set")
		}

		client := testAccProvider.Meta().(*ns1.Client)

		p := rs.Primary

		foundRecord, _, err := client.Records.Get(p.Attributes["zone"], p.Attributes["domain"], p.Attributes["type"])
		if err != nil {
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
	client := testAccProvider.Meta().(*ns1.Client)

	var recordDomain string
	var recordZone string
	var recordType string

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ns1_record" {
			continue
		}

		if rs.Type == "ns1_record" {
			recordType = rs.Primary.Attributes["type"]
			recordDomain = rs.Primary.Attributes["domain"]
			recordZone = rs.Primary.Attributes["zone"]
		}
	}

	foundRecord, _, err := client.Records.Get(recordZone, recordDomain, recordType)
	if err != ns1.ErrRecordMissing {
		return fmt.Errorf("Record still exists: %#v %#v", foundRecord, err)
	}

	return nil
}

func testAccCheckRecordDomain(r *dns.Record, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if r.Domain != expected {
			return fmt.Errorf("Domain: got: %#v want: %#v", r.Domain, expected)
		}
		return nil
	}
}

func testAccCheckRecordTTL(r *dns.Record, expected int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if r.TTL != expected {
			return fmt.Errorf("TTL: got: %#v want: %#v", r.TTL, expected)
		}
		return nil
	}
}

func testAccCheckRecordUseClientSubnet(r *dns.Record, expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *r.UseClientSubnet != expected {
			return fmt.Errorf("UseClientSubnet: got: %#v want: %#v", *r.UseClientSubnet, expected)
		}
		return nil
	}
}

func testAccCheckRecordRegionName(r *dns.Record, expected []string) resource.TestCheckFunc {
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

func testAccCheckRecordAnswerMetaWeight(r *dns.Record, expected float64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		recordAnswer := r.Answers[0]
		recordMetas := recordAnswer.Meta
		weight := recordMetas.Weight.(float64)
		if weight != expected {
			return fmt.Errorf("Answers[0].Meta.Weight: got: %#v want: %#v", weight, expected)
		}
		return nil
	}
}

func testAccCheckRecordAnswerRdata(r *dns.Record, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		recordAnswer := r.Answers[0]
		recordAnswerString := recordAnswer.Rdata[0]
		if recordAnswerString != expected {
			return fmt.Errorf("Answers[0].Rdata[0]: got: %#v want: %#v", recordAnswerString, expected)
		}
		return nil
	}
}

const testAccRecordBasic = `
resource "ns1_record" "it" {
  zone              = "${ns1_zone.test.zone}"
  domain            = "test.${ns1_zone.test.zone}"
  type              = "CNAME"
  ttl               = 60

  // meta {
  //   weight = 5
  //   connections = 3
  //   // up = false // Ignored by d.GetOk("meta.0.up") due to known issue
  // }

  answers {
    answer = "test1.terraform-record-test.io"
    region = "cal"

    // meta {
    //   weight = 10
    //   up = true
    // }
  }

  regions {
    name = "cal"
    // meta {
    //   up = true
    //   us_state = ["CA"]
    // }
  }

  filters {
    filter = "up"
  }

  filters {
    filter = "geotarget_country"
  }

  filters {
    filter = "select_first_n"
    config = {N=1}
  }
}

resource "ns1_zone" "test" {
  zone = "terraform-record-test.io"
}
`

const testAccRecordUpdated = `
resource "ns1_record" "it" {
  zone              = "${ns1_zone.test.zone}"
  domain            = "test.${ns1_zone.test.zone}"
  type              = "CNAME"
  ttl               = 120
  use_client_subnet = false

  // meta {
  //   weight = 5
  //   connections = 3
  //   // up = false // Ignored by d.GetOk("meta.0.up") due to known issue
  // }

  answers {
    answer = "test2.terraform-record-test.io"
    region = "ny"

    // meta {
    //   weight = 5
    //   up = true
    // }
  }

  regions {
    name = "wa"
    // meta {
    //   us_state = ["WA"]
    // }
  }

  regions {
    name = "ny"
    // meta {
    //   us_state = ["NY"]
    // }
  }

  filters {
    filter = "up"
  }

  filters {
    filter = "geotarget_country"
  }
}

resource "ns1_zone" "test" {
  zone = "terraform-record-test.io"
}
`
