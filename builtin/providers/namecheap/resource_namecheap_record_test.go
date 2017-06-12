package namecheap

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/HX-Rd/namecheap"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccNamecheapRecord_Basic(t *testing.T) {
	var record namecheap.Record
	domain := os.Getenv("NAMECHEAP_DOMAIN")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNamecheapRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckNamecheapRecordConfig_basic, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNamecheapRecordExists("namecheap_record.foobar", &record),
					testAccCheckNamecheapRecordAttributes(&record),
					resource.TestCheckResourceAttr(
						"namecheap_record.foobar", "name", "www"),
					resource.TestCheckResourceAttr(
						"namecheap_record.foobar", "domain", domain),
					resource.TestCheckResourceAttr(
						"namecheap_record.foobar", "address", "test.domain."),
				),
			},
		},
	})
}

func TestAccNamecheapRecord_Updated(t *testing.T) {
	var record namecheap.Record
	domain := os.Getenv("NAMECHEAP_DOMAIN")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNamecheapRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckNamecheapRecordConfig_basic, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNamecheapRecordExists("namecheap_record.foobar", &record),
					testAccCheckNamecheapRecordAttributes(&record),
					resource.TestCheckResourceAttr(
						"namecheap_record.foobar", "name", "www"),
					resource.TestCheckResourceAttr(
						"namecheap_record.foobar", "domain", domain),
					resource.TestCheckResourceAttr(
						"namecheap_record.foobar", "address", "test.domain."),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckNamecheapRecordConfig_new_value, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNamecheapRecordExists("namecheap_record.foobar", &record),
					testAccCheckNamecheapRecordAttributesUpdated(&record),
					resource.TestCheckResourceAttr(
						"namecheap_record.foobar", "name", "www"),
					resource.TestCheckResourceAttr(
						"namecheap_record.foobar", "domain", domain),
					resource.TestCheckResourceAttr(
						"namecheap_record.foobar", "address", "test2.domain."),
				),
			},
		},
	})
}

func testAccCheckNamecheapRecordDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*namecheap.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "namecheap_record" {
			continue
		}

		intId, err := strconv.Atoi(rs.Primary.ID)

		if err != nil {
			return fmt.Errorf("Error in converting string id to int id")
		}

		_, err = client.ReadRecord(rs.Primary.Attributes["domain"], intId)

		if err == nil {
			return fmt.Errorf("Record still exists")
		}
	}

	return nil
}

func testAccCheckNamecheapRecordAttributes(record *namecheap.Record) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if record.Address != "test.domain." {
			return fmt.Errorf("Bad address: %s", record.Address)
		}

		return nil
	}
}

func testAccCheckNamecheapRecordAttributesUpdated(record *namecheap.Record) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if record.Address != "test2.domain." {
			return fmt.Errorf("Bad address: %s", record.Address)
		}

		return nil
	}
}

func testAccCheckNamecheapRecordExists(n string, record *namecheap.Record) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*namecheap.Client)

		intId, err := strconv.Atoi(rs.Primary.ID)

		if err != nil {
			return fmt.Errorf("Error in converting string id to int id")
		}

		foundRecord, err := client.ReadRecord(rs.Primary.Attributes["domain"], intId)

		if err != nil {
			return err
		}

		*record = *foundRecord

		return nil
	}
}

const testAccCheckNamecheapRecordConfig_basic = `
resource "namecheap_record" "foobar" {
	domain = "%s"
	name = "www"
	address = "test.domain."
	type = "CNAME"
}`

const testAccCheckNamecheapRecordConfig_new_value = `
resource "namecheap_record" "foobar" {
	domain = "%s"
	name = "www"
	address = "test2.domain."
	type = "CNAME"
}`
