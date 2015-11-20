package infoblox

import (
	"fmt"
	"os"
	"testing"

	"github.com/fanatic/go-infoblox"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccInfobloxRecord_Basic(t *testing.T) {
	var record infoblox.RecordAObject
	domain := os.Getenv("INFOBLOX_DOMAIN")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInfobloxRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckInfobloxRecordConfig_basic, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInfobloxRecordExists("infoblox_record.foobar", &record),
					testAccCheckInfobloxRecordAttributes(&record),
					resource.TestCheckResourceAttr(
						"infoblox_record.foobar", "name", "terraform"),
					resource.TestCheckResourceAttr(
						"infoblox_record.foobar", "domain", domain),
					resource.TestCheckResourceAttr(
						"infoblox_record.foobar", "ipv4addr", "192.168.0.10"),
				),
			},
		},
	})
}

func TestAccInfobloxRecord_Updated(t *testing.T) {
	var record infoblox.RecordAObject
	domain := os.Getenv("INFOBLOX_DOMAIN")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInfobloxRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckInfobloxRecordConfig_basic, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInfobloxRecordExists("infoblox_record.foobar", &record),
					testAccCheckInfobloxRecordAttributes(&record),
					resource.TestCheckResourceAttr(
						"infoblox_record.foobar", "name", "terraform"),
					resource.TestCheckResourceAttr(
						"infoblox_record.foobar", "domain", domain),
					resource.TestCheckResourceAttr(
						"infoblox_record.foobar", "ipv4addr", "192.168.0.10"),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckInfobloxRecordConfig_new_value, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInfobloxRecordExists("infoblox_record.foobar", &record),
					testAccCheckInfobloxRecordAttributesUpdated(&record),
					resource.TestCheckResourceAttr(
						"infoblox_record.foobar", "name", "terraform"),
					resource.TestCheckResourceAttr(
						"infoblox_record.foobar", "domain", domain),
					resource.TestCheckResourceAttr(
						"infoblox_record.foobar", "ipv4addr", "192.168.0.11"),
				),
			},
		},
	})
}

func testAccCheckInfobloxRecordDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*infoblox.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "infoblox_record" {
			continue
		}
		_, err := client.GetRecordA(rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Record still exists")
		}
	}

	return nil
}

func testAccCheckInfobloxRecordAttributes(record *infoblox.RecordAObject) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if record.Ipv4Addr != "192.168.0.10" {
			return fmt.Errorf("Bad content: %s", record.Ipv4Addr)
		}

		return nil
	}
}

func testAccCheckInfobloxRecordAttributesUpdated(record *infoblox.RecordAObject) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if record.Ipv4Addr != "192.168.0.11" {
			return fmt.Errorf("Bad content: %s", record.Ipv4Addr)
		}

		return nil
	}
}

func testAccCheckInfobloxRecordExists(n string, record *infoblox.RecordAObject) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*infoblox.Client)

		foundRecord, err := client.GetRecordA(rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundRecord.Ref != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		*record = *foundRecord

		return nil
	}
}

const testAccCheckInfobloxRecordConfig_basic = `
resource "infoblox_record" "foobar" {
	domain = "%s"

	ipv4addr = "192.168.0.10"
	name = "terraform"
}`

const testAccCheckInfobloxRecordConfig_new_value = `
resource "infoblox_record" "foobar" {
	domain = "%s"

	ipv4addr = "192.168.0.11"
        name = "terraform"
}`
