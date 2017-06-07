package vcd

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccVcdVpn_Basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVcdVpnDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckVcdVpn_basic, os.Getenv("VCD_EDGE_GATEWAY")),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"vcd_edgegateway_vpn.vpn", "encryption_protocol", "AES256"),
				),
			},
		},
	})
}

func testAccCheckVcdVpnDestroy(s *terraform.State) error {

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vcd_edgegateway_vpn" {
			continue
		}

		return nil
	}

	return nil
}

const testAccCheckVcdVpn_basic = `
resource "vcd_edgegateway_vpn" "vpn" {
    edge_gateway        = "%s"
    name                = "west-to-east"
	description         = "Description"
	encryption_protocol = "AES256"
    mtu                 = 1400
    peer_id             = "51.179.218.195"
    peer_ip_address     = "51.179.218.195"
    local_id            = "51.179.218.196"
    local_ip_address    = "51.179.218.196"
    shared_secret       = "yZ4B8pxS5334m6ho692hjbtb7zo2vbesn7pe8ry5hyud86M433tbnnfxt6Dqn73g"
    
    peer_subnets {
        peer_subnet_name = "DMZ_WEST"
        peer_subnet_gateway = "10.0.10.1"
        peer_subnet_mask = "255.255.255.0"
    }

    peer_subnets {
        peer_subnet_name = "WEB_WEST"
        peer_subnet_gateway = "10.0.20.1"
        peer_subnet_mask = "255.255.255.0"
    }

    local_subnets {
        local_subnet_name = "DMZ_EAST"
        local_subnet_gateway = "10.0.1.1"
        local_subnet_mask = "255.255.255.0"
    }

    local_subnets {
        local_subnet_name = "WEB_EAST"
        local_subnet_gateway = "10.0.22.1"
        local_subnet_mask = "255.255.255.0"
    }
}
`
