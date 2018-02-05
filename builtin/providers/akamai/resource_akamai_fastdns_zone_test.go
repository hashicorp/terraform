package akamai

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/configdns-v1"
)

var testAccAkamaiFastDNSZoneConfig = fmt.Sprintf(`
provider "akamai" {
  edgerc = "~/.edgerc"
  fastdns_section = "dns"
}

resource "akamai_fastdns_zone" "tf_acc_test_zone" {
  hostname = "akamaideveloper.net"
  soa {
    ttl = 900
    originserver = "akamaideveloper.net."
    contact = "hostmaster.akamaideveloper.net."
    refresh = 900
    retry = 300
    expire = 604800
    minimum = 180
  }
  a {
    name = "web"
    ttl = 900
    active = true
    target = "1.2.3.4"
  }
  a {
    name = "web"
    ttl = 600
    active = true
    target = "5.6.7.8"
  }
  cname {
    name = "www"
    ttl = 600
    active = true
    target = "blog.akamaideveloper.net."
  }
}
`)

var testAccAkamaiFastDNSZoneConfigWithCounter = fmt.Sprintf(`
provider "akamai" {
  edgerc = "~/.edgerc"
  fastdns_section = "dns"
}

resource "akamai_fastdns_zone" "tf_acc_test_zone_counter" {
  count = "3"
  hostname = "akamaideveloper.net"

  a {
    name = "www${count.index}"
    ttl = 900
    active = true
    target = "1.2.3.4"
  }
}
`)

func TestAccAkamaiFastDNSZone_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAkamaiFastDNSZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAkamaiFastDNSZoneConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAkamaiFastDNSZoneExists,
				),
			},
		},
	})
}

func TestAccAkamaiFastDNSZone_counter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAkamaiFastDNSZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAkamaiFastDNSZoneConfigWithCounter,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAkamaiFastDNSZoneExists,
				),
			},
		},
	})
}

func testAccCheckAkamaiFastDNSZoneDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "akamai_fastdns_zone" {
			continue
		}

		hostname := strings.Split(rs.Primary.ID, "-")[2]
		zone, err := dns.GetZone(hostname)
		if err != nil {
			return err
		}
		if len(zone.Zone.A) > 0 ||
			len(zone.Zone.Aaaa) > 0 ||
			len(zone.Zone.Afsdb) > 0 ||
			len(zone.Zone.Cname) > 0 ||
			len(zone.Zone.Dnskey) > 0 ||
			len(zone.Zone.Ds) > 0 ||
			len(zone.Zone.Hinfo) > 0 ||
			len(zone.Zone.Loc) > 0 ||
			len(zone.Zone.Mx) > 0 ||
			len(zone.Zone.Naptr) > 0 ||
			len(zone.Zone.Nsec3) > 0 ||
			len(zone.Zone.Nsec3param) > 0 ||
			len(zone.Zone.Ptr) > 0 ||
			len(zone.Zone.Rp) > 0 ||
			len(zone.Zone.Rrsig) > 0 ||
			len(zone.Zone.Spf) > 0 ||
			len(zone.Zone.Srv) > 0 ||
			len(zone.Zone.Sshfp) > 0 ||
			len(zone.Zone.Txt) > 0 {
			// These never get deleted
			// len(zone.Zone.Ns) > 0 ||
			// len(zone.Zone.Soa) > 0 ||
			return fmt.Errorf("Zone was not deleted %s", zone)
		}
	}
	return nil
}

func testAccCheckAkamaiFastDNSZoneExists(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "akamai_fastdns_zone" {
			continue
		}

		hostname := strings.Split(rs.Primary.ID, "-")[2]
		_, err := dns.GetZone(hostname)
		if err != nil {
			return err
		}
	}
	return nil
}
