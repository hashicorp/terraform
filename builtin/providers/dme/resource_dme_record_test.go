package dme

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/soniah/dnsmadeeasy"
)

var _ = fmt.Sprintf("dummy") // dummy
var _ = os.DevNull           // dummy

func TestAccDMERecord_basic(t *testing.T) {
	var record dnsmadeeasy.Record
	domainid := os.Getenv("DME_DOMAINID")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDMERecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testDMERecordConfigA, domainid),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDMERecordExists("dme_record.test", &record),
					resource.TestCheckResourceAttr(
						"dme_record.test", "domainid", domainid),
					resource.TestCheckResourceAttr(
						"dme_record.test", "name", "testa"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "type", "A"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "value", "1.1.1.1"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "ttl", "2000"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "gtdLocation", "DEFAULT"),
				),
			},
		},
	})
}

func TestAccDMERecordCName(t *testing.T) {
	var record dnsmadeeasy.Record
	domainid := os.Getenv("DME_DOMAINID")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDMERecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testDMERecordConfigCName, domainid),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDMERecordExists("dme_record.test", &record),
					resource.TestCheckResourceAttr(
						"dme_record.test", "domainid", domainid),
					resource.TestCheckResourceAttr(
						"dme_record.test", "name", "testcname"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "type", "CNAME"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "value", "foo"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "ttl", "2000"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "gtdLocation", "DEFAULT"),
				),
			},
		},
	})
}

func TestAccDMERecordMX(t *testing.T) {
	var record dnsmadeeasy.Record
	domainid := os.Getenv("DME_DOMAINID")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDMERecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testDMERecordConfigMX, domainid),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDMERecordExists("dme_record.test", &record),
					resource.TestCheckResourceAttr(
						"dme_record.test", "domainid", domainid),
					resource.TestCheckResourceAttr(
						"dme_record.test", "name", "testmx"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "type", "MX"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "value", "foo"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "mxLevel", "10"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "ttl", "2000"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "gtdLocation", "DEFAULT"),
				),
			},
		},
	})
}

func TestAccDMERecordHTTPRED(t *testing.T) {
	var record dnsmadeeasy.Record
	domainid := os.Getenv("DME_DOMAINID")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDMERecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testDMERecordConfigHTTPRED, domainid),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDMERecordExists("dme_record.test", &record),
					resource.TestCheckResourceAttr(
						"dme_record.test", "domainid", domainid),
					resource.TestCheckResourceAttr(
						"dme_record.test", "name", "testhttpred"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "type", "HTTPRED"),

					resource.TestCheckResourceAttr(
						"dme_record.test", "value", "https://github.com/soniah/terraform-provider-dme"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "hardLink", "true"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "redirectType", "Hidden Frame Masked"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "title", "An Example"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "keywords", "terraform example"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "description", "This is a description"),

					resource.TestCheckResourceAttr(
						"dme_record.test", "ttl", "2000"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "gtdLocation", "DEFAULT"),
				),
			},
		},
	})
}

func TestAccDMERecordTXT(t *testing.T) {
	var record dnsmadeeasy.Record
	domainid := os.Getenv("DME_DOMAINID")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDMERecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testDMERecordConfigTXT, domainid),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDMERecordExists("dme_record.test", &record),
					resource.TestCheckResourceAttr(
						"dme_record.test", "domainid", domainid),
					resource.TestCheckResourceAttr(
						"dme_record.test", "name", "testtxt"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "type", "TXT"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "value", "\"foo\""),
					resource.TestCheckResourceAttr(
						"dme_record.test", "ttl", "2000"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "gtdLocation", "DEFAULT"),
				),
			},
		},
	})
}

func TestAccDMERecordSPF(t *testing.T) {
	var record dnsmadeeasy.Record
	domainid := os.Getenv("DME_DOMAINID")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDMERecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testDMERecordConfigSPF, domainid),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDMERecordExists("dme_record.test", &record),
					resource.TestCheckResourceAttr(
						"dme_record.test", "domainid", domainid),
					resource.TestCheckResourceAttr(
						"dme_record.test", "name", "testspf"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "type", "SPF"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "value", "\"foo\""),
					resource.TestCheckResourceAttr(
						"dme_record.test", "ttl", "2000"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "gtdLocation", "DEFAULT"),
				),
			},
		},
	})
}

func TestAccDMERecordPTR(t *testing.T) {
	var record dnsmadeeasy.Record
	domainid := os.Getenv("DME_DOMAINID")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDMERecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testDMERecordConfigPTR, domainid),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDMERecordExists("dme_record.test", &record),
					resource.TestCheckResourceAttr(
						"dme_record.test", "domainid", domainid),
					resource.TestCheckResourceAttr(
						"dme_record.test", "name", "testptr"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "type", "PTR"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "value", "foo"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "ttl", "2000"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "gtdLocation", "DEFAULT"),
				),
			},
		},
	})
}

func TestAccDMERecordNS(t *testing.T) {
	var record dnsmadeeasy.Record
	domainid := os.Getenv("DME_DOMAINID")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDMERecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testDMERecordConfigNS, domainid),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDMERecordExists("dme_record.test", &record),
					resource.TestCheckResourceAttr(
						"dme_record.test", "domainid", domainid),
					resource.TestCheckResourceAttr(
						"dme_record.test", "name", "testns"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "type", "NS"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "value", "foo"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "ttl", "2000"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "gtdLocation", "DEFAULT"),
				),
			},
		},
	})
}

func TestAccDMERecordAAAA(t *testing.T) {
	var record dnsmadeeasy.Record
	domainid := os.Getenv("DME_DOMAINID")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDMERecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testDMERecordConfigAAAA, domainid),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDMERecordExists("dme_record.test", &record),
					resource.TestCheckResourceAttr(
						"dme_record.test", "domainid", domainid),
					resource.TestCheckResourceAttr(
						"dme_record.test", "name", "testaaaa"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "type", "AAAA"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "value", "fe80::0202:b3ff:fe1e:8329"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "ttl", "2000"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "gtdLocation", "DEFAULT"),
				),
			},
		},
	})
}

func TestAccDMERecordSRV(t *testing.T) {
	var record dnsmadeeasy.Record
	domainid := os.Getenv("DME_DOMAINID")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDMERecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testDMERecordConfigSRV, domainid),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDMERecordExists("dme_record.test", &record),
					resource.TestCheckResourceAttr(
						"dme_record.test", "domainid", domainid),
					resource.TestCheckResourceAttr(
						"dme_record.test", "name", "testsrv"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "type", "SRV"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "value", "foo"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "priority", "10"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "weight", "20"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "port", "30"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "ttl", "2000"),
					resource.TestCheckResourceAttr(
						"dme_record.test", "gtdLocation", "DEFAULT"),
				),
			},
		},
	})
}

func testAccCheckDMERecordDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*dnsmadeeasy.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "dnsmadeeasy_record" {
			continue
		}

		_, err := client.ReadRecord(rs.Primary.Attributes["domainid"], rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Record still exists")
		}
	}

	return nil
}

func testAccCheckDMERecordExists(n string, record *dnsmadeeasy.Record) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*dnsmadeeasy.Client)

		foundRecord, err := client.ReadRecord(rs.Primary.Attributes["domainid"], rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundRecord.StringRecordID() != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		*record = *foundRecord

		return nil
	}
}

const testDMERecordConfigA = `
resource "dme_record" "test" {
  domainid = "%s"
  name = "testa"
  type = "A"
  value = "1.1.1.1"
  ttl = 2000
  gtdLocation = "DEFAULT"
}`

const testDMERecordConfigCName = `
resource "dme_record" "test" {
  domainid = "%s"
  name = "testcname"
  type = "CNAME"
  value = "foo"
  ttl = 2000
  gtdLocation = "DEFAULT"
}`

const testDMERecordConfigAName = `
resource "dme_record" "test" {
  domainid = "%s"
  name = "testaname"
  type = "ANAME"
  value = "foo"
  ttl = 2000
  gtdLocation = "DEFAULT"
}`

const testDMERecordConfigMX = `
resource "dme_record" "test" {
  domainid = "%s"
  name = "testmx"
  type = "MX"
  value = "foo"
  mxLevel = 10
  ttl = 2000
  gtdLocation = "DEFAULT"
}`

const testDMERecordConfigHTTPRED = `
resource "dme_record" "test" {
  domainid = "%s"
  name = "testhttpred"
  type = "HTTPRED"
  value = "https://github.com/soniah/terraform-provider-dme"
  hardLink = true
  redirectType = "Hidden Frame Masked"
  title = "An Example"
  keywords = "terraform example"
  description = "This is a description"
  ttl = 2000
  gtdLocation = "DEFAULT"
}`

const testDMERecordConfigTXT = `
resource "dme_record" "test" {
  domainid = "%s"
  name = "testtxt"
  type = "TXT"
  value = "foo"
  ttl = 2000
  gtdLocation = "DEFAULT"
}`

const testDMERecordConfigSPF = `
resource "dme_record" "test" {
  domainid = "%s"
  name = "testspf"
  type = "SPF"
  value = "foo"
  ttl = 2000
  gtdLocation = "DEFAULT"
}`

const testDMERecordConfigPTR = `
resource "dme_record" "test" {
  domainid = "%s"
  name = "testptr"
  type = "PTR"
  value = "foo"
  ttl = 2000
  gtdLocation = "DEFAULT"
}`

const testDMERecordConfigNS = `
resource "dme_record" "test" {
  domainid = "%s"
  name = "testns"
  type = "NS"
  value = "foo"
  ttl = 2000
  gtdLocation = "DEFAULT"
}`

const testDMERecordConfigAAAA = `
resource "dme_record" "test" {
  domainid = "%s"
  name = "testaaaa"
  type = "AAAA"
  value = "FE80::0202:B3FF:FE1E:8329"
  ttl = 2000
  gtdLocation = "DEFAULT"
}`

const testDMERecordConfigSRV = `
resource "dme_record" "test" {
  domainid = "%s"
  name = "testsrv"
  type = "SRV"
  value = "foo"
  priority = 10
  weight = 20
  port = 30
  ttl = 2000
  gtdLocation = "DEFAULT"
}`
