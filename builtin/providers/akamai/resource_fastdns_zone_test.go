package akamai

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

var testAccAkamaiFastDNSZoneConfig = fmt.Sprintf(`
provider "akamai" {
  edgerc = "/Users/Johanna/.edgerc"
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
    name = "wwwq"
    ttl = 600
    active = true
    target = "blog.akamaideveloper.net."
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

func testAccCheckAkamaiFastDNSZoneDestroy(s *terraform.State) error {
	return nil
}

func testAccCheckAkamaiFastDNSZoneExists(s *terraform.State) error {
	return nil
}
