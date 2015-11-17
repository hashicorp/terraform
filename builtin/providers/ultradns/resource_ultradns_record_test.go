package ultradns

import (
	"fmt"
	"os"
	"testing"

	"github.com/Ensighten/udnssdk"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccUltraDNSRecord_Basic(t *testing.T) {
	var record udnssdk.RRSet
	domain := os.Getenv("ULTRADNS_DOMAIN")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckUltraDNSRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckUltraDNSRecordConfig_basic, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUltraDNSRecordExists("ultradns_record.foobar", &record),
					testAccCheckUltraDNSRecordAttributes(&record),
					resource.TestCheckResourceAttr(
						"ultradns_record.foobar", "name", "terraform"),
					resource.TestCheckResourceAttr(
						"ultradns_record.foobar", "zone", domain),
					resource.TestCheckResourceAttr(
						"ultradns_record.foobar", "rdata.0", "192.168.0.10"),
				),
			},
		},
	})
}

func TestAccUltraDNSRecord_Updated(t *testing.T) {
	var record udnssdk.RRSet
	domain := os.Getenv("ULTRADNS_DOMAIN")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckUltraDNSRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckUltraDNSRecordConfig_basic, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUltraDNSRecordExists("ultradns_record.foobar", &record),
					testAccCheckUltraDNSRecordAttributes(&record),
					resource.TestCheckResourceAttr(
						"ultradns_record.foobar", "name", "terraform"),
					resource.TestCheckResourceAttr(
						"ultradns_record.foobar", "zone", domain),
					resource.TestCheckResourceAttr(
						"ultradns_record.foobar", "rdata.0", "192.168.0.10"),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckUltraDNSRecordConfig_new_value, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUltraDNSRecordExists("ultradns_record.foobar", &record),
					testAccCheckUltraDNSRecordAttributesUpdated(&record),
					resource.TestCheckResourceAttr(
						"ultradns_record.foobar", "name", "terraform"),
					resource.TestCheckResourceAttr(
						"ultradns_record.foobar", "zone", domain),
					resource.TestCheckResourceAttr(
						"ultradns_record.foobar", "rdata.0", "192.168.0.11"),
				),
			},
		},
	})
}

func testAccCheckUltraDNSRecordDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*udnssdk.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ultradns_record" {
			continue
		}

		_, _, err := client.RRSets.GetRRSets(rs.Primary.Attributes["zone"], rs.Primary.Attributes["name"], rs.Primary.Attributes["type"])

		if err == nil {
			return fmt.Errorf("Record still exists")
		}
	}

	return nil
}

func testAccCheckUltraDNSRecordAttributes(record *udnssdk.RRSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if record.RData[0] != "192.168.0.10" {
			return fmt.Errorf("Bad content: %v", record.RData)
		}

		return nil
	}
}

func testAccCheckUltraDNSRecordAttributesUpdated(record *udnssdk.RRSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if record.RData[0] != "192.168.0.11" {
			return fmt.Errorf("Bad content: %v", record.RData)
		}

		return nil
	}
}

func testAccCheckUltraDNSRecordExists(n string, record *udnssdk.RRSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*udnssdk.Client)
		foundRecord, _, err := client.RRSets.GetRRSets(rs.Primary.Attributes["zone"], rs.Primary.Attributes["name"], rs.Primary.Attributes["type"])

		if err != nil {
			return err
		}

		if foundRecord[0].OwnerName != rs.Primary.Attributes["hostname"] {
			return fmt.Errorf("Record not found: %+v,\n %+v\n", foundRecord, rs.Primary.Attributes)
		}

		*record = foundRecord[0]

		return nil
	}
}

const testAccCheckUltraDNSRecordConfig_basic = `
resource "ultradns_record" "foobar" {
	zone = "%s"

	name = "terraform"
	rdata = [ "192.168.0.10" ]
	type = "A"
	ttl = 3600
}`

const testAccCheckUltraDNSRecordConfig_new_value = `
resource "ultradns_record" "foobar" {
	zone = "%s"

	name = "terraform"
	rdata = [ "192.168.0.11" ]
	type = "A"
	ttl = 3600
}`
