package dns

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/miekg/dns"
)

func TestAccDnsCnameRecord_basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDnsCnameRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDnsCnameRecord_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDnsCnameRecordExists(t, "dns_cname_record.foo", "bar.example.com."),
				),
			},
			resource.TestStep{
				Config: testAccDnsCnameRecord_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDnsCnameRecordExists(t, "dns_cname_record.foo", "baz.example.com."),
				),
			},
		},
	})
}

func testAccCheckDnsCnameRecordDestroy(s *terraform.State) error {
	meta := testAccProvider.Meta()
	c := meta.(*DNSClient).c
	srv_addr := meta.(*DNSClient).srv_addr
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "dns_cname_record" {
			continue
		}

		rec_name := rs.Primary.Attributes["name"]
		rec_zone := rs.Primary.Attributes["zone"]

		if rec_zone != dns.Fqdn(rec_zone) {
			return fmt.Errorf("Error reading DNS record: \"zone\" should be an FQDN")
		}

		rec_fqdn := fmt.Sprintf("%s.%s", rec_name, rec_zone)

		msg := new(dns.Msg)
		msg.SetQuestion(rec_fqdn, dns.TypeCNAME)
		r, _, err := c.Exchange(msg, srv_addr)
		if err != nil {
			return fmt.Errorf("Error querying DNS record: %s", err)
		}
		if r.Rcode != dns.RcodeNameError {
			return fmt.Errorf("DNS record still exists: %v", r.Rcode)
		}
	}

	return nil
}

func testAccCheckDnsCnameRecordExists(t *testing.T, n string, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		rec_name := rs.Primary.Attributes["name"]
		rec_zone := rs.Primary.Attributes["zone"]

		if rec_zone != dns.Fqdn(rec_zone) {
			return fmt.Errorf("Error reading DNS record: \"zone\" should be an FQDN")
		}

		rec_fqdn := fmt.Sprintf("%s.%s", rec_name, rec_zone)

		meta := testAccProvider.Meta()
		c := meta.(*DNSClient).c
		srv_addr := meta.(*DNSClient).srv_addr

		msg := new(dns.Msg)
		msg.SetQuestion(rec_fqdn, dns.TypeCNAME)
		r, _, err := c.Exchange(msg, srv_addr)
		if err != nil {
			return fmt.Errorf("Error querying DNS record: %s", err)
		}
		if r.Rcode != dns.RcodeSuccess {
			return fmt.Errorf("Error querying DNS record")
		}

		if len(r.Answer) > 1 {
			return fmt.Errorf("Error querying DNS record: multiple responses received")
		}
		record := r.Answer[0]
		cname, err := getCnameVal(record)
		if err != nil {
			return fmt.Errorf("Error querying DNS record: %s", err)
		}
		if expected != cname {
			return fmt.Errorf("DNS record differs: expected %v, found %v", expected, cname)
		}
		return nil
	}
}

var testAccDnsCnameRecord_basic = fmt.Sprintf(`
  resource "dns_cname_record" "foo" {
    zone = "example.com."
    name = "foo"
    cname = "bar.example.com."
    ttl = 300
  }`)

var testAccDnsCnameRecord_update = fmt.Sprintf(`
  resource "dns_cname_record" "foo" {
    zone = "example.com."
    name = "baz"
    cname = "baz.example.com."
    ttl = 300
  }`)
