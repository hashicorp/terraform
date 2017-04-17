package dns

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/miekg/dns"
)

func TestAccDnsAAAARecordSet_basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDnsAAAARecordSetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDnsAAAARecordSet_basic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("dns_aaaa_record_set.bar", "addresses.#", "2"),
					testAccCheckDnsAAAARecordSetExists(t, "dns_aaaa_record_set.bar", []interface{}{"fdd5:e282:43b8:5303:dead:beef:cafe:babe", "fdd5:e282:43b8:5303:cafe:babe:dead:beef"}),
				),
			},
			resource.TestStep{
				Config: testAccDnsAAAARecordSet_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("dns_aaaa_record_set.bar", "addresses.#", "2"),
					testAccCheckDnsAAAARecordSetExists(t, "dns_aaaa_record_set.bar", []interface{}{"fdd5:e282:43b8:5303:beef:dead:babe:cafe", "fdd5:e282:43b8:5303:babe:cafe:beef:dead"}),
				),
			},
		},
	})
}

func testAccCheckDnsAAAARecordSetDestroy(s *terraform.State) error {
	meta := testAccProvider.Meta()
	c := meta.(*DNSClient).c
	srv_addr := meta.(*DNSClient).srv_addr
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "dns_aaaa_record_set" {
			continue
		}

		rec_name := rs.Primary.Attributes["name"]
		rec_zone := rs.Primary.Attributes["zone"]

		if rec_zone != dns.Fqdn(rec_zone) {
			return fmt.Errorf("Error reading DNS record: \"zone\" should be an FQDN")
		}

		rec_fqdn := fmt.Sprintf("%s.%s", rec_name, rec_zone)

		msg := new(dns.Msg)
		msg.SetQuestion(rec_fqdn, dns.TypeAAAA)
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

func testAccCheckDnsAAAARecordSetExists(t *testing.T, n string, addr []interface{}) resource.TestCheckFunc {
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
		msg.SetQuestion(rec_fqdn, dns.TypeAAAA)
		r, _, err := c.Exchange(msg, srv_addr)
		if err != nil {
			return fmt.Errorf("Error querying DNS record: %s", err)
		}
		if r.Rcode != dns.RcodeSuccess {
			return fmt.Errorf("Error querying DNS record")
		}

		addresses := schema.NewSet(schema.HashString, nil)
		expected := schema.NewSet(schema.HashString, addr)
		for _, record := range r.Answer {
			addr, err := getAAAAVal(record)
			if err != nil {
				return fmt.Errorf("Error querying DNS record: %s", err)
			}
			addresses.Add(addr)
		}
		if !addresses.Equal(expected) {
			return fmt.Errorf("DNS record differs: expected %v, found %v", expected, addresses)
		}
		return nil
	}
}

var testAccDnsAAAARecordSet_basic = fmt.Sprintf(`
  resource "dns_aaaa_record_set" "bar" {
    zone = "example.com."
    name = "bar"
    addresses = ["fdd5:e282:43b8:5303:dead:beef:cafe:babe", "fdd5:e282:43b8:5303:cafe:babe:dead:beef"]
    ttl = 300
  }`)

var testAccDnsAAAARecordSet_update = fmt.Sprintf(`
  resource "dns_aaaa_record_set" "bar" {
    zone = "example.com."
    name = "bar"
    addresses = ["fdd5:e282:43b8:5303:beef:dead:babe:cafe", "fdd5:e282:43b8:5303:babe:cafe:beef:dead"]
    ttl = 300
  }`)
