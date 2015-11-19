package digitalocean

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDigitalOceanRecord_Basic(t *testing.T) {
	var record godo.DomainRecord

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
	var record godo.DomainRecord

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

func TestAccDigitalOceanRecord_HostnameValue(t *testing.T) {
	var record godo.DomainRecord

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDigitalOceanRecordConfig_cname,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanRecordExists("digitalocean_record.foobar", &record),
					testAccCheckDigitalOceanRecordAttributesHostname("a", &record),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "name", "terraform"),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "domain", "foobar-test-terraform.com"),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "value", "a.foobar-test-terraform.com."),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "type", "CNAME"),
				),
			},
		},
	})
}

// This test fails with:
//
//   POST https://api.digitalocean.com/v2/domains/foobar-test-terraform.com/records:
//   422 Data needs to end with a dot (.)
//
// Which seems like a behavior change on the DO API side. Opened support ticket
// #826791 to ask DigitalOcean about this, and we'll comment out the test for
// now. --phinze
/*
func TestAccDigitalOceanRecord_RelativeHostnameValue(t *testing.T) {
	var record godo.DomainRecord

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDigitalOceanRecordConfig_relative_cname,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanRecordExists("digitalocean_record.foobar", &record),
					testAccCheckDigitalOceanRecordAttributesHostname("a.b", &record),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "name", "terraform"),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "domain", "foobar-test-terraform.com"),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "value", "a.b"),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "type", "CNAME"),
				),
			},
		},
	})
}
*/

func TestAccDigitalOceanRecord_ExternalHostnameValue(t *testing.T) {
	var record godo.DomainRecord

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDigitalOceanRecordConfig_external_cname,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanRecordExists("digitalocean_record.foobar", &record),
					testAccCheckDigitalOceanRecordAttributesHostname("a.foobar-test-terraform.net", &record),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "name", "terraform"),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "domain", "foobar-test-terraform.com"),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "value", "a.foobar-test-terraform.net."),
					resource.TestCheckResourceAttr(
						"digitalocean_record.foobar", "type", "CNAME"),
				),
			},
		},
	})
}

func testAccCheckDigitalOceanRecordDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*godo.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "digitalocean_record" {
			continue
		}
		domain := rs.Primary.Attributes["domain"]
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}

		_, _, err = client.Domains.Record(domain, id)

		if err == nil {
			return fmt.Errorf("Record still exists")
		}
	}

	return nil
}

func testAccCheckDigitalOceanRecordAttributes(record *godo.DomainRecord) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if record.Data != "192.168.0.10" {
			return fmt.Errorf("Bad value: %s", record.Data)
		}

		return nil
	}
}

func testAccCheckDigitalOceanRecordAttributesUpdated(record *godo.DomainRecord) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if record.Data != "192.168.0.11" {
			return fmt.Errorf("Bad value: %s", record.Data)
		}

		return nil
	}
}

func testAccCheckDigitalOceanRecordExists(n string, record *godo.DomainRecord) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*godo.Client)

		domain := rs.Primary.Attributes["domain"]
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}

		foundRecord, _, err := client.Domains.Record(domain, id)

		if err != nil {
			return err
		}

		if strconv.Itoa(foundRecord.ID) != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		*record = *foundRecord

		return nil
	}
}

func testAccCheckDigitalOceanRecordAttributesHostname(data string, record *godo.DomainRecord) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if record.Data != data {
			return fmt.Errorf("Bad value: expected %s, got %s", data, record.Data)
		}

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

const testAccCheckDigitalOceanRecordConfig_cname = `
resource "digitalocean_domain" "foobar" {
    name = "foobar-test-terraform.com"
    ip_address = "192.168.0.10"
}

resource "digitalocean_record" "foobar" {
    domain = "${digitalocean_domain.foobar.name}"

    name = "terraform"
    value = "a.foobar-test-terraform.com."
    type = "CNAME"
}`

const testAccCheckDigitalOceanRecordConfig_relative_cname = `
resource "digitalocean_domain" "foobar" {
    name = "foobar-test-terraform.com"
    ip_address = "192.168.0.10"
}

resource "digitalocean_record" "foobar" {
    domain = "${digitalocean_domain.foobar.name}"

    name = "terraform"
    value = "a.b"
    type = "CNAME"
}`

const testAccCheckDigitalOceanRecordConfig_external_cname = `
resource "digitalocean_domain" "foobar" {
    name = "foobar-test-terraform.com"
    ip_address = "192.168.0.10"
}

resource "digitalocean_record" "foobar" {
    domain = "${digitalocean_domain.foobar.name}"

    name = "terraform"
    value = "a.foobar-test-terraform.net."
    type = "CNAME"
}`
