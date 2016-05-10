package ultradns

import (
	"fmt"
	"testing"

	"github.com/Ensighten/udnssdk"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccUltraDNSRecord_Basic(t *testing.T) {
	var record udnssdk.RRSet
	// domain := os.Getenv("ULTRADNS_DOMAIN")
	domain := "ultradns.phinze.com"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckUltraDNSRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckUltraDNSRecordConfigBasic, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUltraDNSRecordExists("ultradns_record.basic", &record),
					testAccCheckUltraDNSRecordAttributes(&record),
					resource.TestCheckResourceAttr(
						"ultradns_record.basic", "name", "basic"),
					resource.TestCheckResourceAttr(
						"ultradns_record.basic", "zone", domain),
					resource.TestCheckResourceAttr(
						"ultradns_record.basic", "rdata.0", "192.168.0.10"),
				),
			},
		},
	})
}

func TestAccUltraDNSRecord_Updated(t *testing.T) {
	var record udnssdk.RRSet
	domain := "ultradns.phinze.com"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckUltraDNSRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckUltraDNSRecordConfigBasic, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUltraDNSRecordExists("ultradns_record.basic", &record),
					testAccCheckUltraDNSRecordAttributes(&record),
					resource.TestCheckResourceAttr(
						"ultradns_record.basic", "name", "basic"),
					resource.TestCheckResourceAttr(
						"ultradns_record.basic", "zone", domain),
					resource.TestCheckResourceAttr(
						"ultradns_record.basic", "rdata.0", "192.168.0.10"),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckUltraDNSRecordConfigNewValue, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUltraDNSRecordExists("ultradns_record.basic", &record),
					testAccCheckUltraDNSRecordAttributesUpdated(&record),
					resource.TestCheckResourceAttr(
						"ultradns_record.basic", "name", "basic"),
					resource.TestCheckResourceAttr(
						"ultradns_record.basic", "zone", domain),
					resource.TestCheckResourceAttr(
						"ultradns_record.basic", "rdata.0", "192.168.0.11"),
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

		k := udnssdk.RRSetKey{
			Zone: rs.Primary.Attributes["zone"],
			Name: rs.Primary.Attributes["name"],
			Type: rs.Primary.Attributes["type"],
		}

		_, err := client.RRSets.Select(k)

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
		k := udnssdk.RRSetKey{
			Zone: rs.Primary.Attributes["zone"],
			Name: rs.Primary.Attributes["name"],
			Type: rs.Primary.Attributes["type"],
		}

		foundRecord, err := client.RRSets.Select(k)

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

const testAccCheckUltraDNSRecordConfigBasic = `
resource "ultradns_record" "basic" {
  zone = "%s"

  name  = "basic"
  rdata = ["192.168.0.10"]
  type  = "A"
  ttl   = 3600
}
`

const testAccCheckUltraDNSRecordConfigNewValue = `
resource "ultradns_record" "basic" {
  zone = "%s"
  name  = "basic"

  rdata = ["192.168.0.11"]
  type  = "A"
  ttl   = 3600
}
`
