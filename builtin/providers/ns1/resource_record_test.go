package ns1

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/dns"
)

func TestAccNS1Record_Basic(t *testing.T) {
	var record dns.Record
	zone := fmt.Sprintf("terraform.acctest-%s.com", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNS1RecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccNS1Record_basic, zone),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNS1RecordExists("ns1_record.foobar", &record),
					testAccCheckNS1RecordAttributes(&record, zone),
					resource.TestCheckResourceAttr("ns1_record.foobar", "domain", "test."+zone),
					resource.TestCheckResourceAttr("ns1_record.foobar", "type", "CNAME"),
					resource.TestCheckResourceAttr("ns1_record.foobar", "ttl", "300"),
					resource.TestCheckResourceAttr("ns1_record.foobar", "answer.#", "2"),
					resource.TestCheckResourceAttr("ns1_record.foobar", "answer.0.rdata.0", "test1."+zone),
					resource.TestCheckResourceAttr("ns1_record.foobar", "answer.1.rdata.0", "test2."+zone),
				),
			},
		},
	})
}

func TestAccNS1Record_Updated(t *testing.T) {
	var record dns.Record
	zone := fmt.Sprintf("terraform.acctest-%s.com", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNS1RecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccNS1Record_basic, zone),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNS1RecordExists("ns1_record.foobar", &record),
					testAccCheckNS1RecordAttributes(&record, zone),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testAccNS1Record_updated, zone),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNS1RecordExists("ns1_record.foobar", &record),
					testAccCheckNS1RecordAttributesUpdated(&record, zone),
					resource.TestCheckResourceAttr("ns1_record.foobar", "ttl", "120"),
					resource.TestCheckResourceAttr("ns1_record.foobar", "use_client_subnet", "false"),
					resource.TestCheckResourceAttr("ns1_record.foobar", "answer.#", "3"),
					resource.TestCheckResourceAttr("ns1_record.foobar", "answer.0.rdata.0", "updated."+zone),
					resource.TestCheckResourceAttr("ns1_record.foobar", "answer.1.rdata.0", "test2."+zone),
					resource.TestCheckResourceAttr("ns1_record.foobar", "answer.2.rdata.0", "test3."+zone),
				),
			},
		},
	})
}

func testAccCheckNS1RecordExists(n string, record *dns.Record) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("NoID is set")
		}

		client := testAccProvider.Meta().(*ns1.Client)

		foundRecord, _, err := client.Records.Get(
			rs.Primary.Attributes["zone"],
			rs.Primary.Attributes["domain"],
			rs.Primary.Attributes["type"],
		)

		if err != nil {
			return err
		}

		if foundRecord.ID != rs.Primary.Attributes["id"] {
			return fmt.Errorf("Record not found")
		}

		*record = *foundRecord

		return nil
	}
}

func testAccCheckNS1RecordDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ns1.Client)

	var zone, domain, t string
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ns1_record" {
			continue
		}

		if rs.Type == "ns1_record" {
			zone = rs.Primary.Attributes["zone"]
			domain = rs.Primary.Attributes["domain"]
			t = rs.Primary.Attributes["type"]

			_, _, err := client.Records.Get(zone, domain, t)

			if err != ns1.ErrRecordMissing {
				return fmt.Errorf("Record still exists: %s %s %s %s", err, zone, domain, t)
			}
		}
	}

	return nil
}

func testAccCheckNS1RecordAttributes(record *dns.Record, zone string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if record.Zone != zone {
			return fmt.Errorf("Bad value record.Zone: %s", record.Zone)
		}
		if record.Domain != "test."+zone {
			return fmt.Errorf("Bad value record.Domain: %s", record.Domain)
		}
		if record.Type != "CNAME" {
			return fmt.Errorf("Bad value record.Type: %s", record.Type)
		}
		if record.TTL != 300 {
			return fmt.Errorf("Bad value record.TTL: %d", record.TTL)
		}
		if *record.UseClientSubnet != true {
			return fmt.Errorf("Bad value record.UseClientSubnet: %t", record.UseClientSubnet)
		}

		if len(record.Answers) != 2 {
			return fmt.Errorf("Wrong number of answers: %d", len(record.Answers))
		}

		ans1 := record.Answers[0]
		if ans1.Rdata[0] != "test1."+zone {
			return fmt.Errorf("Bad value record.Answers[0].Rdata: %v", record.Answers[0].Rdata)
		}
		ans2 := record.Answers[1]
		if ans2.Rdata[0] != "test2."+zone {
			return fmt.Errorf("Bad value record.Answers[1].Rdata: %v", record.Answers[1].Rdata)
		}

		return nil
	}
}

func testAccCheckNS1RecordAttributesUpdated(record *dns.Record, zone string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if record.Zone != zone {
			return fmt.Errorf("Bad value record.Zone: %s", record.Zone)
		}
		if record.Domain != "test."+zone {
			return fmt.Errorf("Bad value record.Domain: %s", record.Domain)
		}
		if record.Type != "CNAME" {
			return fmt.Errorf("Bad value record.Type: %s", record.Type)
		}
		if record.TTL != 120 {
			return fmt.Errorf("Bad value record.TTL: %d", record.TTL)
		}
		if *record.UseClientSubnet != false {
			return fmt.Errorf("Bad value record.UseClientSubnet: %t", record.UseClientSubnet)
		}

		if len(record.Answers) != 3 {
			return fmt.Errorf("Wrong number of answers: %d", len(record.Answers))
		}

		ans1 := record.Answers[0]
		if ans1.Rdata[0] != "updated."+zone {
			return fmt.Errorf("Bad value record.Answers[0].Rdata: %v", record.Answers[0].Rdata)
		}
		ans2 := record.Answers[1]
		if ans2.Rdata[0] != "test2."+zone {
			return fmt.Errorf("Bad value record.Answers[1].Rdata: %v", record.Answers[1].Rdata)
		}
		ans3 := record.Answers[2]
		if ans3.Rdata[0] != "test3."+zone {
			return fmt.Errorf("Bad value record.Answers[2].Rdata: %v", record.Answers[2].Rdata)
		}

		return nil
	}
}

const testAccNS1Record_basic = `
resource "ns1_zone" "test" {
    zone = "%s"
}
resource "ns1_record" "foobar" {
  zone = "${ns1_zone.test.zone}"
  domain = "test.${ns1_zone.test.zone}"
  type = "CNAME"
  ttl = 300

  answer {
    rdata = ["test1.${ns1_zone.test.zone}"]
  }
  answer {
    rdata = ["test2.${ns1_zone.test.zone}"]
  }

  filter {
    type = "select_first_n"
    config = {N=1}
  }
}`

const testAccNS1Record_updated = `
resource "ns1_zone" "test" {
    zone = "%s"
}
resource "ns1_record" "foobar" {
  zone = "${ns1_zone.test.zone}"
  domain = "test.${ns1_zone.test.zone}"
  type = "CNAME"
  ttl = 120
  use_client_subnet = false

  answer {
    rdata = ["updated.${ns1_zone.test.zone}"]
  }
  answer {
    rdata = ["test2.${ns1_zone.test.zone}"]
  }
  answer {
    rdata = ["test3.${ns1_zone.test.zone}"]
  }

  filter {
    type = "select_first_n"
    config = {N=1}
  }
}`
