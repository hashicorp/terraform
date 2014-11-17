package dnsimple

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/pearkes/dnsimple"
)

func TestAccDNSimpleRecord_Basic(t *testing.T) {
	var record dnsimple.Record
	domain := os.Getenv("DNSIMPLE_DOMAIN")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDNSimpleRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckDNSimpleRecordConfig_basic, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDNSimpleRecordExists("dnsimple_record.foobar", &record),
					testAccCheckDNSimpleRecordAttributes(&record),
					resource.TestCheckResourceAttr(
						"dnsimple_record.foobar", "name", "terraform"),
					resource.TestCheckResourceAttr(
						"dnsimple_record.foobar", "domain", domain),
					resource.TestCheckResourceAttr(
						"dnsimple_record.foobar", "value", "192.168.0.10"),
				),
			},
		},
	})
}

func TestAccDNSimpleRecord_Updated(t *testing.T) {
	var record dnsimple.Record
	domain := os.Getenv("DNSIMPLE_DOMAIN")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDNSimpleRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckDNSimpleRecordConfig_basic, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDNSimpleRecordExists("dnsimple_record.foobar", &record),
					testAccCheckDNSimpleRecordAttributes(&record),
					resource.TestCheckResourceAttr(
						"dnsimple_record.foobar", "name", "terraform"),
					resource.TestCheckResourceAttr(
						"dnsimple_record.foobar", "domain", domain),
					resource.TestCheckResourceAttr(
						"dnsimple_record.foobar", "value", "192.168.0.10"),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckDNSimpleRecordConfig_new_value, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDNSimpleRecordExists("dnsimple_record.foobar", &record),
					testAccCheckDNSimpleRecordAttributesUpdated(&record),
					resource.TestCheckResourceAttr(
						"dnsimple_record.foobar", "name", "terraform"),
					resource.TestCheckResourceAttr(
						"dnsimple_record.foobar", "domain", domain),
					resource.TestCheckResourceAttr(
						"dnsimple_record.foobar", "value", "192.168.0.11"),
				),
			},
		},
	})
}

func testAccCheckDNSimpleRecordDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*dnsimple.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "dnsimple_record" {
			continue
		}

		_, err := client.RetrieveRecord(rs.Primary.Attributes["domain"], rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Record still exists")
		}
	}

	return nil
}

func testAccCheckDNSimpleRecordAttributes(record *dnsimple.Record) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if record.Content != "192.168.0.10" {
			return fmt.Errorf("Bad content: %s", record.Content)
		}

		return nil
	}
}

func testAccCheckDNSimpleRecordAttributesUpdated(record *dnsimple.Record) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if record.Content != "192.168.0.11" {
			return fmt.Errorf("Bad content: %s", record.Content)
		}

		return nil
	}
}

func testAccCheckDNSimpleRecordExists(n string, record *dnsimple.Record) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*dnsimple.Client)

		foundRecord, err := client.RetrieveRecord(rs.Primary.Attributes["domain"], rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundRecord.StringId() != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		*record = *foundRecord

		return nil
	}
}

const testAccCheckDNSimpleRecordConfig_basic = `
resource "dnsimple_record" "foobar" {
	domain = "%s"

	name = "terraform"
	value = "192.168.0.10"
	type = "A"
	ttl = 3600
}`

const testAccCheckDNSimpleRecordConfig_new_value = `
resource "dnsimple_record" "foobar" {
	domain = "%s"

	name = "terraform"
	value = "192.168.0.11"
	type = "A"
	ttl = 3600
}`
