package dns

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/miekg/dns"
)

func TestAccDnsARecordSet_Basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDnsARecordSetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDnsARecordSet_basic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("dns_a_record_set.foo", "addresses.#", "2"),
					testAccCheckDnsARecordSetExists(t, "dns_a_record_set.foo", []interface{}{"192.168.0.2", "192.168.0.1"}),
				),
			},
			resource.TestStep{
				Config: testAccDnsARecordSet_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("dns_a_record_set.foo", "addresses.#", "3"),
					testAccCheckDnsARecordSetExists(t, "dns_a_record_set.foo", []interface{}{"10.0.0.3", "10.0.0.2", "10.0.0.1"}),
				),
			},
		},
	})
}

func testAccCheckDnsARecordSetDestroy(s *terraform.State) error {
	meta := testAccProvider.Meta()
	c := meta.(*DNSClient).c
	srv_addr := meta.(*DNSClient).srv_addr
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "dns_a_record_set" {
			continue
		}

		rec_name := rs.Primary.Attributes["name"]
		rec_zone := rs.Primary.Attributes["zone"]

		if rec_zone != dns.Fqdn(rec_zone) {
			return fmt.Errorf("Error reading DNS record: \"zone\" should be an FQDN")
		}

		rec_fqdn := fmt.Sprintf("%s.%s", rec_name, rec_zone)

		msg := new(dns.Msg)
		msg.SetQuestion(rec_fqdn, dns.TypeA)
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

func testAccCheckDnsARecordSetExists(t *testing.T, n string, addr []interface{}) resource.TestCheckFunc {
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
		msg.SetQuestion(rec_fqdn, dns.TypeA)
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
			addr, err := getAVal(record)
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

var testAccDnsARecordSet_basic = fmt.Sprintf(`
  resource "dns_a_record_set" "foo" {
    zone = "example.com."
    name = "foo"
    addresses = ["192.168.0.1", "192.168.0.2"]
    ttl = 300
  }`)

var testAccDnsARecordSet_update = fmt.Sprintf(`
  resource "dns_a_record_set" "foo" {
    zone = "example.com."
    name = "foo"
    addresses = ["10.0.0.1", "10.0.0.2", "10.0.0.3"]
    ttl = 300
  }`)
