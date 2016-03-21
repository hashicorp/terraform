package clc

import (
	"fmt"
	"testing"

	clc "github.com/CenturyLinkCloud/clc-sdk"
	"github.com/CenturyLinkCloud/clc-sdk/server"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

// things to test:
//   maps to internal specified ip
//   port range
//   update existing rule
//   CIDR restriction

func TestAccPublicIPBasic(t *testing.T) {
	var resp server.PublicIP
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPublicIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckPublicIPConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPublicIPExists("clc_public_ip.acc_test_public_ip", &resp),
					testAccCheckPublicIPNIC("clc_public_ip.acc_test_public_ip", &resp),
					testAccCheckPublicIPPortRange("clc_public_ip.acc_test_public_ip", &resp),
					testAccCheckPublicIPBlockCIDR("clc_public_ip.acc_test_public_ip", &resp),
					//testAccCheckPublicIPUpdated("clc_public_ip.eip", &resp),
				),
			},
		},
	})
}

func testAccCheckPublicIPDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*clc.Client)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "clc_public_ip" {
			continue
		}
		sid := rs.Primary.Attributes["server_id"]
		_, err := client.Server.GetPublicIP(sid, rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("IP still exists")
		}
	}
	return nil

}

func testAccCheckPublicIPExists(n string, resp *server.PublicIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No PublicIP ID is set")
		}
		client := testAccProvider.Meta().(*clc.Client)
		sid := rs.Primary.Attributes["server_id"]
		p, err := client.Server.GetPublicIP(sid, rs.Primary.ID)
		if err != nil {
			return err
		}
		*resp = *p
		return nil
	}
}

func testAccCheckPublicIPPortRange(n string, resp *server.PublicIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// check the passed port range made it through
		var spec server.Port
		for _, p := range resp.Ports {
			if p.Protocol == "UDP" {
				spec = p
				break
			}
		}
		if spec.Port != 53 || spec.PortTo != 55 {
			return fmt.Errorf("Expected udp ports from 53-55 but found: %v", spec)
		}
		return nil
	}
}
func testAccCheckPublicIPBlockCIDR(n string, resp *server.PublicIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// check the passed port range made it through
		spec := resp.SourceRestrictions[0]
		if spec.CIDR != "108.19.67.15/32" {
			return fmt.Errorf("Expected cidr restriction but found: %v", spec)
		}
		return nil
	}
}

func testAccCheckPublicIPNIC(n string, resp *server.PublicIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		sid := rs.Primary.Attributes["server_id"]
		nic := rs.Primary.Attributes["internal_ip_address"]

		client := testAccProvider.Meta().(*clc.Client)
		srv, err := client.Server.Get(sid)
		if err != nil {
			return fmt.Errorf("Failed fetching server? %v", err)
		}
		first := srv.Details.IPaddresses[0].Internal
		if nic != first {
			return fmt.Errorf("Expected public ip to be mapped to %s but found: %s", first, nic)
		}
		return nil
	}
}

var testAccCheckPublicIPConfigBasic = `
variable "dc" { default = "IL1" }

resource "clc_group" "acc_test_group_ip" {
  location_id		= "${var.dc}"
  name			= "acc_test_group_ip"
  parent		= "Default Group"
}

resource "clc_server" "acc_test_server" {
  name_template		= "test"
  source_server_id	= "UBUNTU-14-64-TEMPLATE"
  group_id		= "${clc_group.acc_test_group_ip.id}"
  cpu			= 1
  memory_mb		= 1024
  password		= "Green123$"
}

resource "clc_public_ip" "acc_test_public_ip" {
  server_id		= "${clc_server.acc_test_server.id}"
  internal_ip_address	= "${clc_server.acc_test_server.private_ip_address}"
  source_restrictions
     { cidr		= "108.19.67.15/32" }
  ports
    {
      protocol		= "TCP"
      port		= 80
    }
  ports
    {
      protocol		= "UDP"
      port		= 53
      port_to		= 55
    }
}
`
