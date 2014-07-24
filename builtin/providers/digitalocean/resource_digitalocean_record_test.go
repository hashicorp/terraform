package digitalocean

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/pearkes/digitalocean"
)

func TestAccDigitalOceanRecord_Basic(t *testing.T) {
	var record digitalocean.Record

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDigitalOceanRecordConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanRecordExists("digitalocean_record.foobar", &record),
					testAccCheckDigitalOceanRecordAttributes(&record),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "name", "terraform"),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "domain", "foobar-test-terraform.com"),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "value", "192.168.0.10"),
				),
			},
		},
	})
}

func TestAccDigitalOceanRecord_Updated(t *testing.T) {
	var record digitalocean.Record

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDigitalOceanRecordConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanRecordExists("digitalocean_record.foobar", &record),
					testAccCheckDigitalOceanRecordAttributes(&record),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "name", "terraform"),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "domain", "foobar-test-terraform.com"),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "value", "192.168.0.10"),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "type", "A"),
				),
			},
			resource.TestStep{
				Config: testAccCheckDigitalOceanRecordConfig_new_value,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanRecordExists("digitalocean_record.foobar", &record),
					testAccCheckDigitalOceanRecordAttributesUpdated(&record),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "name", "terraform"),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "domain", "foobar-test-terraform.com"),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "value", "192.168.0.11"),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "type", "A"),
				),
			},
		},
	})
}

func testAccCheckDigitalOceanRecordDestroy(s *terraform.State) error {
	client := testAccProvider.client

	for _, rs := range s.Resources {
		if rs.Type != "digitalocean_record" {
			continue
		}

		_, err := client.RetrieveRecord(rs.Attributes["domain"], rs.ID)

		if err == nil {
			return fmt.Errorf("Record still exists")
		}
	}

	return nil
}

func testAccCheckDigitalOceanRecordAttributes(record *digitalocean.Record) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if record.Data != "192.168.0.10" {
			return fmt.Errorf("Bad value: %s", record.Data)
		}

		return nil
	}
}

func testAccCheckDigitalOceanRecordAttributesUpdated(record *digitalocean.Record) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if record.Data != "192.168.0.11" {
			return fmt.Errorf("Bad value: %s", record.Data)
		}

		return nil
	}
}

func testAccCheckDigitalOceanRecordExists(n string, record *digitalocean.Record) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.client

		foundRecord, err := client.RetrieveRecord(rs.Attributes["domain"], rs.ID)

		if err != nil {
			return err
		}

		if foundRecord.StringId() != rs.ID {
			return fmt.Errorf("Record not found")
		}

		*record = foundRecord

		return nil
	}
}

const testAccCheckDigitalOceanRecordConfig_basic = `
resource "digitalocean_domain" "foobar" {
    name = "foobar-test-terraform.com"
    ip_address = "192.168.0.10"
}

resource "digitalocean_record" "foobar" {
    domain = "${digitalocean_domain.foobar.name}"

    name = "terraform"
    value = "192.168.0.10"
    type = "A"
}`

const testAccCheckDigitalOceanRecordConfig_new_value = `
resource "digitalocean_domain" "foobar" {
    name = "foobar-test-terraform.com"
    ip_address = "192.168.0.10"
}

resource "digitalocean_record" "foobar" {
    domain = "${digitalocean_domain.foobar.name}"

    name = "terraform"
    value = "192.168.0.11"
    type = "A"
}`
