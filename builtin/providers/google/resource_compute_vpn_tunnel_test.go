package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"google.golang.org/api/compute/v1"
)

func TestAccComputeVpnTunnel_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeVpnTunnelDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeVpnTunnel_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeVpnTunnelExists(
						"google_compute_vpn_tunnel.foobar"),
					resource.TestCheckResourceAttr(
						"google_compute_vpn_tunnel.foobar", "local_traffic_selector.#", "1"),
					resource.TestCheckResourceAttr(
						"google_compute_vpn_tunnel.foobar", "remote_traffic_selector.#", "2"),
				),
			},
		},
	})
}

func TestAccComputeVpnTunnel_router(t *testing.T) {
	router := fmt.Sprintf("tunnel-test-router-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeVpnTunnelDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeVpnTunnelRouter(router),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeVpnTunnelExists(
						"google_compute_vpn_tunnel.foobar"),
					resource.TestCheckResourceAttr(
						"google_compute_vpn_tunnel.foobar", "router", router),
				),
			},
		},
	})
}

func TestAccComputeVpnTunnel_defaultTrafficSelectors(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeVpnTunnelDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeVpnTunnelDefaultTrafficSelectors,
				Check: testAccCheckComputeVpnTunnelExists(
					"google_compute_vpn_tunnel.foobar"),
			},
		},
	})
}

func testAccCheckComputeVpnTunnelDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	project := config.Project

	vpnTunnelsService := compute.NewVpnTunnelsService(config.clientCompute)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_network" {
			continue
		}

		region := rs.Primary.Attributes["region"]
		name := rs.Primary.Attributes["name"]

		_, err := vpnTunnelsService.Get(project, region, name).Do()

		if err == nil {
			return fmt.Errorf("Error, VPN Tunnel %s in region %s still exists",
				name, region)
		}
	}

	return nil
}

func testAccCheckComputeVpnTunnelExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)
		name := rs.Primary.Attributes["name"]
		region := rs.Primary.Attributes["region"]
		project := config.Project

		vpnTunnelsService := compute.NewVpnTunnelsService(config.clientCompute)
		_, err := vpnTunnelsService.Get(project, region, name).Do()

		if err != nil {
			return fmt.Errorf("Error Reading VPN Tunnel %s: %s", name, err)
		}

		return nil
	}
}

var testAccComputeVpnTunnel_basic = fmt.Sprintf(`
resource "google_compute_network" "foobar" {
	name = "tunnel-test-%s"
}
resource "google_compute_subnetwork" "foobar" {
	name = "tunnel-test-%s"
	network = "${google_compute_network.foobar.self_link}"
	ip_cidr_range = "10.0.0.0/16"
	region = "us-central1"
}
resource "google_compute_address" "foobar" {
	name = "tunnel-test-%s"
	region = "${google_compute_subnetwork.foobar.region}"
}
resource "google_compute_vpn_gateway" "foobar" {
	name = "tunnel-test-%s"
	network = "${google_compute_network.foobar.self_link}"
	region = "${google_compute_subnetwork.foobar.region}"
}
resource "google_compute_forwarding_rule" "foobar_esp" {
	name = "tunnel-test-%s"
	region = "${google_compute_vpn_gateway.foobar.region}"
	ip_protocol = "ESP"
	ip_address = "${google_compute_address.foobar.address}"
	target = "${google_compute_vpn_gateway.foobar.self_link}"
}
resource "google_compute_forwarding_rule" "foobar_udp500" {
	name = "tunnel-test-%s"
	region = "${google_compute_forwarding_rule.foobar_esp.region}"
	ip_protocol = "UDP"
	port_range = "500-500"
	ip_address = "${google_compute_address.foobar.address}"
	target = "${google_compute_vpn_gateway.foobar.self_link}"
}
resource "google_compute_forwarding_rule" "foobar_udp4500" {
	name = "tunnel-test-%s"
	region = "${google_compute_forwarding_rule.foobar_udp500.region}"
	ip_protocol = "UDP"
	port_range = "4500-4500"
	ip_address = "${google_compute_address.foobar.address}"
	target = "${google_compute_vpn_gateway.foobar.self_link}"
}
resource "google_compute_vpn_tunnel" "foobar" {
	name = "tunnel-test-%s"
	region = "${google_compute_forwarding_rule.foobar_udp4500.region}"
	target_vpn_gateway = "${google_compute_vpn_gateway.foobar.self_link}"
	shared_secret = "unguessable"
	peer_ip = "8.8.8.8"
	local_traffic_selector = ["${google_compute_subnetwork.foobar.ip_cidr_range}"]
	remote_traffic_selector = ["192.168.0.0/24", "192.168.1.0/24"]
}`, acctest.RandString(10), acctest.RandString(10), acctest.RandString(10),
	acctest.RandString(10), acctest.RandString(10), acctest.RandString(10),
	acctest.RandString(10), acctest.RandString(10))

func testAccComputeVpnTunnelRouter(router string) string {
	testId := acctest.RandString(10)
	return fmt.Sprintf(`
		resource "google_compute_network" "foobar" {
			name = "tunnel-test-%s"
		}
		resource "google_compute_subnetwork" "foobar" {
			name = "tunnel-test-%s"
			network = "${google_compute_network.foobar.self_link}"
			ip_cidr_range = "10.0.0.0/16"
			region = "us-central1"
		}
		resource "google_compute_address" "foobar" {
			name = "tunnel-test-%s"
			region = "${google_compute_subnetwork.foobar.region}"
		}
		resource "google_compute_vpn_gateway" "foobar" {
			name = "tunnel-test-%s"
			network = "${google_compute_network.foobar.self_link}"
			region = "${google_compute_subnetwork.foobar.region}"
		}
		resource "google_compute_forwarding_rule" "foobar_esp" {
			name = "tunnel-test-%s-1"
			region = "${google_compute_vpn_gateway.foobar.region}"
			ip_protocol = "ESP"
			ip_address = "${google_compute_address.foobar.address}"
			target = "${google_compute_vpn_gateway.foobar.self_link}"
		}
		resource "google_compute_forwarding_rule" "foobar_udp500" {
			name = "tunnel-test-%s-2"
			region = "${google_compute_forwarding_rule.foobar_esp.region}"
			ip_protocol = "UDP"
			port_range = "500-500"
			ip_address = "${google_compute_address.foobar.address}"
			target = "${google_compute_vpn_gateway.foobar.self_link}"
		}
		resource "google_compute_forwarding_rule" "foobar_udp4500" {
			name = "tunnel-test-%s-3"
			region = "${google_compute_forwarding_rule.foobar_udp500.region}"
			ip_protocol = "UDP"
			port_range = "4500-4500"
			ip_address = "${google_compute_address.foobar.address}"
			target = "${google_compute_vpn_gateway.foobar.self_link}"
		}
		resource "google_compute_router" "foobar"{
			name = "%s"
			region = "${google_compute_forwarding_rule.foobar_udp500.region}"
			network = "${google_compute_network.foobar.self_link}"
			bgp {
				asn = 64514
			}
		}
		resource "google_compute_vpn_tunnel" "foobar" {
			name = "tunnel-test-%s"
			region = "${google_compute_forwarding_rule.foobar_udp4500.region}"
			target_vpn_gateway = "${google_compute_vpn_gateway.foobar.self_link}"
			shared_secret = "unguessable"
			peer_ip = "8.8.8.8"
			router = "${google_compute_router.foobar.name}"
		}
	`, testId, testId, testId, testId, testId, testId, testId, router, testId)
}

var testAccComputeVpnTunnelDefaultTrafficSelectors = fmt.Sprintf(`
resource "google_compute_network" "foobar" {
	name = "tunnel-test-%s"
	auto_create_subnetworks = "true"
}
resource "google_compute_address" "foobar" {
	name = "tunnel-test-%s"
	region = "us-central1"
}
resource "google_compute_vpn_gateway" "foobar" {
	name = "tunnel-test-%s"
	network = "${google_compute_network.foobar.self_link}"
	region = "${google_compute_address.foobar.region}"
}
resource "google_compute_forwarding_rule" "foobar_esp" {
	name = "tunnel-test-%s"
	region = "${google_compute_vpn_gateway.foobar.region}"
	ip_protocol = "ESP"
	ip_address = "${google_compute_address.foobar.address}"
	target = "${google_compute_vpn_gateway.foobar.self_link}"
}
resource "google_compute_forwarding_rule" "foobar_udp500" {
	name = "tunnel-test-%s"
	region = "${google_compute_forwarding_rule.foobar_esp.region}"
	ip_protocol = "UDP"
	port_range = "500-500"
	ip_address = "${google_compute_address.foobar.address}"
	target = "${google_compute_vpn_gateway.foobar.self_link}"
}
resource "google_compute_forwarding_rule" "foobar_udp4500" {
	name = "tunnel-test-%s"
	region = "${google_compute_forwarding_rule.foobar_udp500.region}"
	ip_protocol = "UDP"
	port_range = "4500-4500"
	ip_address = "${google_compute_address.foobar.address}"
	target = "${google_compute_vpn_gateway.foobar.self_link}"
}
resource "google_compute_vpn_tunnel" "foobar" {
	name = "tunnel-test-%s"
	region = "${google_compute_forwarding_rule.foobar_udp4500.region}"
	target_vpn_gateway = "${google_compute_vpn_gateway.foobar.self_link}"
	shared_secret = "unguessable"
	peer_ip = "8.8.8.8"
}`, acctest.RandString(10), acctest.RandString(10), acctest.RandString(10),
	acctest.RandString(10), acctest.RandString(10), acctest.RandString(10),
	acctest.RandString(10))
