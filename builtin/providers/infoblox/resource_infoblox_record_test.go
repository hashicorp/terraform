package infoblox

import (
	"fmt"
	"os"
	"strings"
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
				Config: fmt.Sprintf(testInfobloxRecordConfigA, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInfobloxRecordAExists("infoblox_record.test", &record),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "name", "testa"),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "domain", domain),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "type", "A"),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "value", "10.1.1.43"),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "ttl", "600"),
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
				Config: fmt.Sprintf(testInfobloxRecordConfigA, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInfobloxRecordAExists("infoblox_record.test", &record),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "name", "testa"),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "domain", domain),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "type", "A"),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "value", "10.1.1.43"),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "ttl", "600"),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testInfobloxRecordConfigANew, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInfobloxRecordAExists("infoblox_record.test", &record),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "name", "testa"),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "domain", domain),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "type", "A"),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "value", "10.1.1.50"),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "ttl", "600"),
				),
			},
		},
	})
}

func TestAccInfobloxRecordAAAA(t *testing.T) {
	var record infoblox.RecordAAAAObject
	domain := os.Getenv("INFOBLOX_DOMAIN")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInfobloxRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testInfobloxRecordConfigAAAA, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInfobloxRecordAAAAExists("infoblox_record.test", &record),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "name", "testaaaa"),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "domain", domain),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "type", "AAAA"),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "value", "fe80::c634:6bff:fe73:da10"),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "ttl", "600"),
				),
			},
		},
	})
}

func TestAccInfobloxRecordCname(t *testing.T) {
	var record infoblox.RecordCnameObject
	domain := os.Getenv("INFOBLOX_DOMAIN")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInfobloxRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testInfobloxRecordConfigCname, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInfobloxRecordCnameExists("infoblox_record.test", &record),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "name", "testcnamealias"),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "domain", domain),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "type", "CNAME"),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "value", "testcname"),
					resource.TestCheckResourceAttr(
						"infoblox_record.test", "ttl", "600"),
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
		var err error

		switch strings.ToUpper(rs.Primary.Attributes["type"]) {
		case "A":
			_, err = client.GetRecordA(rs.Primary.ID)
		case "AAAA":
			_, err = client.GetRecordAAAA(rs.Primary.ID)
		case "CNAME":
			_, err = client.GetRecordCname(rs.Primary.ID)
		}

		if err == nil {
			return fmt.Errorf("Record still exists")
		}
	}

	return nil
}

func testAccCheckInfobloxRecordAExists(n string, record *infoblox.RecordAObject) resource.TestCheckFunc {
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

func testAccCheckInfobloxRecordAAAAExists(n string, record *infoblox.RecordAAAAObject) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*infoblox.Client)
		foundRecord, err := client.GetRecordAAAA(rs.Primary.ID)

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

func testAccCheckInfobloxRecordCnameExists(n string, record *infoblox.RecordCnameObject) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*infoblox.Client)
		foundRecord, err := client.GetRecordCname(rs.Primary.ID)

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

const testInfobloxRecordConfigA = `
resource "infoblox_record" "test" {
	domain = "%s"
	value = "10.1.1.43"
	name = "testa"
	type = "A"
	ttl = 600

}`

const testInfobloxRecordConfigANew = `
resource "infoblox_record" "test" {
	domain = "%s"
	value = "10.1.1.50"
	name = "testa"
	type = "A"
	ttl = 600

}`

const testInfobloxRecordConfigCname = `
resource "infoblox_record" "test" {
	domain = "%s"
	value = "testcname"
	name = "testcnamealias"
	type = "CNAME"
	ttl = 600

}`

const testInfobloxRecordConfigAAAA = `
resource "infoblox_record" "test" {
	domain = "%s"
	value = "fe80::c634:6bff:fe73:da10"
	name = "testaaaa"
	type = "AAAA"
	ttl = 600

}`
