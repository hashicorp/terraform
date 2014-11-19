package cloudflare

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/pearkes/cloudflare"
)

func TestAccCLOudflareRecord_Basic(t *testing.T) {
	var record cloudflare.Record
	domain := os.Getenv("CLOUDFLARE_DOMAIN")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCLOudflareRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckCLoudFlareRecordConfig_basic, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCLOudflareRecordExists("cloudflare_record.foobar", &record),
					testAccCheckCLOudflareRecordAttributes(&record),
					resource.TestCheckResourceAttr(
						"cloudflare_record.foobar", "name", "terraform"),
					resource.TestCheckResourceAttr(
						"cloudflare_record.foobar", "domain", domain),
					resource.TestCheckResourceAttr(
						"cloudflare_record.foobar", "value", "192.168.0.10"),
				),
			},
		},
	})
}

func TestAccCLOudflareRecord_Updated(t *testing.T) {
	var record cloudflare.Record
	domain := os.Getenv("CLOUDFLARE_DOMAIN")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCLOudflareRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckCLoudFlareRecordConfig_basic, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCLOudflareRecordExists("cloudflare_record.foobar", &record),
					testAccCheckCLOudflareRecordAttributes(&record),
					resource.TestCheckResourceAttr(
						"cloudflare_record.foobar", "name", "terraform"),
					resource.TestCheckResourceAttr(
						"cloudflare_record.foobar", "domain", domain),
					resource.TestCheckResourceAttr(
						"cloudflare_record.foobar", "value", "192.168.0.10"),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckCloudFlareRecordConfig_new_value, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCLOudflareRecordExists("cloudflare_record.foobar", &record),
					testAccCheckCLOudflareRecordAttributesUpdated(&record),
					resource.TestCheckResourceAttr(
						"cloudflare_record.foobar", "name", "terraform"),
					resource.TestCheckResourceAttr(
						"cloudflare_record.foobar", "domain", domain),
					resource.TestCheckResourceAttr(
						"cloudflare_record.foobar", "value", "192.168.0.11"),
				),
			},
		},
	})
}

func testAccCheckCLOudflareRecordDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*cloudflare.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudflare_record" {
			continue
		}

		_, err := client.RetrieveRecord(rs.Primary.Attributes["domain"], rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Record still exists")
		}
	}

	return nil
}

func testAccCheckCLOudflareRecordAttributes(record *cloudflare.Record) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if record.Value != "192.168.0.10" {
			return fmt.Errorf("Bad value: %s", record.Value)
		}

		return nil
	}
}

func testAccCheckCLOudflareRecordAttributesUpdated(record *cloudflare.Record) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if record.Value != "192.168.0.11" {
			return fmt.Errorf("Bad value: %s", record.Value)
		}

		return nil
	}
}

func testAccCheckCLOudflareRecordExists(n string, record *cloudflare.Record) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*cloudflare.Client)

		foundRecord, err := client.RetrieveRecord(rs.Primary.Attributes["domain"], rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundRecord.Id != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		*record = *foundRecord

		return nil
	}
}

const testAccCheckCLoudFlareRecordConfig_basic = `
resource "cloudflare_record" "foobar" {
	domain = "%s"

	name = "terraform"
	value = "192.168.0.10"
	type = "A"
	ttl = 3600
}`

const testAccCheckCloudFlareRecordConfig_new_value = `
resource "cloudflare_record" "foobar" {
	domain = "%s"

	name = "terraform"
	value = "192.168.0.11"
	type = "A"
	ttl = 3600
}`
