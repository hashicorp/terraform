package powerdns

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPDNSRecord_A(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPDNSRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSRecordConfigA,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPDNSRecordExists("powerdns_record.test-a"),
				),
			},
		},
	})
}

func TestAccPDNSRecord_WithCount(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPDNSRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSRecordConfigHyphenedWithCount,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPDNSRecordExists("powerdns_record.test-counted.0"),
					testAccCheckPDNSRecordExists("powerdns_record.test-counted.1"),
				),
			},
		},
	})
}

func TestAccPDNSRecord_AAAA(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPDNSRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSRecordConfigAAAA,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPDNSRecordExists("powerdns_record.test-aaaa"),
				),
			},
		},
	})
}

func TestAccPDNSRecord_CNAME(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPDNSRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSRecordConfigCNAME,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPDNSRecordExists("powerdns_record.test-cname"),
				),
			},
		},
	})
}

func TestAccPDNSRecord_HINFO(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPDNSRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSRecordConfigHINFO,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPDNSRecordExists("powerdns_record.test-hinfo"),
				),
			},
		},
	})
}

func TestAccPDNSRecord_LOC(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPDNSRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSRecordConfigLOC,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPDNSRecordExists("powerdns_record.test-loc"),
				),
			},
		},
	})
}

func TestAccPDNSRecord_MX(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPDNSRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSRecordConfigMX,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPDNSRecordExists("powerdns_record.test-mx"),
				),
			},
			{
				Config: testPDNSRecordConfigMXMulti,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPDNSRecordExists("powerdns_record.test-mx-multi"),
				),
			},
		},
	})
}

func TestAccPDNSRecord_NAPTR(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPDNSRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSRecordConfigNAPTR,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPDNSRecordExists("powerdns_record.test-naptr"),
				),
			},
		},
	})
}

func TestAccPDNSRecord_NS(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPDNSRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSRecordConfigNS,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPDNSRecordExists("powerdns_record.test-ns"),
				),
			},
		},
	})
}

func TestAccPDNSRecord_SPF(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPDNSRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSRecordConfigSPF,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPDNSRecordExists("powerdns_record.test-spf"),
				),
			},
		},
	})
}

func TestAccPDNSRecord_SSHFP(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPDNSRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSRecordConfigSSHFP,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPDNSRecordExists("powerdns_record.test-sshfp"),
				),
			},
		},
	})
}

func TestAccPDNSRecord_SRV(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPDNSRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSRecordConfigSRV,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPDNSRecordExists("powerdns_record.test-srv"),
				),
			},
		},
	})
}

func TestAccPDNSRecord_TXT(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPDNSRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSRecordConfigTXT,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPDNSRecordExists("powerdns_record.test-txt"),
				),
			},
		},
	})
}

func testAccCheckPDNSRecordDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "powerdns_record" {
			continue
		}

		client := testAccProvider.Meta().(*Client)
		exists, err := client.RecordExistsByID(rs.Primary.Attributes["zone"], rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error checking if record still exists: %#v", rs.Primary.ID)
		}
		if exists {
			return fmt.Errorf("Record still exists: %#v", rs.Primary.ID)
		}

	}
	return nil
}

func testAccCheckPDNSRecordExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*Client)
		foundRecords, err := client.ListRecordsByID(rs.Primary.Attributes["zone"], rs.Primary.ID)
		if err != nil {
			return err
		}
		if len(foundRecords) == 0 {
			return fmt.Errorf("Record does not exist")
		}
		for _, rec := range foundRecords {
			if rec.Id() == rs.Primary.ID {
				return nil
			}
		}
		return fmt.Errorf("Record does not exist: %#v", rs.Primary.ID)
	}
}

const testPDNSRecordConfigA = `
resource "powerdns_record" "test-a" {
  zone = "sysa.xyz"
	name = "redis.sysa.xyz"
	type = "A"
	ttl = 60
	records = [ "1.1.1.1", "2.2.2.2" ]
}`

const testPDNSRecordConfigHyphenedWithCount = `
resource "powerdns_record" "test-counted" {
	count = "2"
	zone = "sysa.xyz"
	name = "redis-${count.index}.sysa.xyz"
	type = "A"
	ttl = 60
	records = [ "1.1.1.${count.index}" ]
}`

const testPDNSRecordConfigAAAA = `
resource "powerdns_record" "test-aaaa" {
  zone = "sysa.xyz"
	name = "redis.sysa.xyz"
	type = "AAAA"
	ttl = 60
	records = [ "2001:DB8:2000:bf0::1", "2001:DB8:2000:bf1::1" ]
}`

const testPDNSRecordConfigCNAME = `
resource "powerdns_record" "test-cname" {
  zone = "sysa.xyz"
	name = "redis.sysa.xyz"
	type = "CNAME"
	ttl = 60
	records = [ "redis.example.com" ]
}`

const testPDNSRecordConfigHINFO = `
resource "powerdns_record" "test-hinfo" {
  zone = "sysa.xyz"
	name = "redis.sysa.xyz"
	type = "HINFO"
	ttl = 60
	records = [ "\"PC-Intel-2.4ghz\" \"Linux\"" ]
}`

const testPDNSRecordConfigLOC = `
resource "powerdns_record" "test-loc" {
  zone = "sysa.xyz"
	name = "redis.sysa.xyz"
	type = "LOC"
	ttl = 60
	records = [ "51 56 0.123 N 5 54 0.000 E 4.00m 1.00m 10000.00m 10.00m" ]
}`

const testPDNSRecordConfigMX = `
resource "powerdns_record" "test-mx" {
  zone = "sysa.xyz"
	name = "sysa.xyz"
	type = "MX"
	ttl = 60
	records = [ "10 mail.example.com" ]
}`

const testPDNSRecordConfigMXMulti = `
resource "powerdns_record" "test-mx-multi" {
  zone = "sysa.xyz"
	name = "sysa.xyz"
	type = "MX"
	ttl = 60
	records = [ "10 mail1.example.com", "20 mail2.example.com" ]
}`

const testPDNSRecordConfigNAPTR = `
resource "powerdns_record" "test-naptr" {
  zone = "sysa.xyz"
	name = "sysa.xyz"
	type = "NAPTR"
	ttl = 60
	records = [ "100 50 \"s\" \"z3950+I2L+I2C\" \"\" _z3950._tcp.gatech.edu'." ]
}`

const testPDNSRecordConfigNS = `
resource "powerdns_record" "test-ns" {
  zone = "sysa.xyz"
	name = "lab.sysa.xyz"
	type = "NS"
	ttl = 60
	records = [ "ns1.sysa.xyz", "ns2.sysa.xyz" ]
}`

const testPDNSRecordConfigSPF = `
resource "powerdns_record" "test-spf" {
  zone = "sysa.xyz"
	name = "sysa.xyz"
	type = "SPF"
	ttl = 60
	records = [ "\"v=spf1 +all\"" ]
}`

const testPDNSRecordConfigSSHFP = `
resource "powerdns_record" "test-sshfp" {
  zone = "sysa.xyz"
	name = "ssh.sysa.xyz"
	type = "SSHFP"
	ttl = 60
	records = [ "1 1 123456789abcdef67890123456789abcdef67890" ]
}`

const testPDNSRecordConfigSRV = `
resource "powerdns_record" "test-srv" {
  zone = "sysa.xyz"
	name = "_redis._tcp.sysa.xyz"
	type = "SRV"
	ttl = 60
	records = [ "0 10 6379 redis1.sysa.xyz", "0 10 6379 redis2.sysa.xyz", "10 10 6379 redis-replica.sysa.xyz" ]
}`

const testPDNSRecordConfigTXT = `
resource "powerdns_record" "test-txt" {
  zone = "sysa.xyz"
	name = "text.sysa.xyz"
	type = "TXT"
	ttl = 60
	records = [ "\"text record payload\"" ]
}`
