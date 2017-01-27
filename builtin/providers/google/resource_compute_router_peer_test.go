package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"google.golang.org/api/compute/v1"
)

func TestAccComputeRouterPeer_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeRouterPeer_basic,
				Check: testAccCheckComputeRouterPeerExists(
					"google_compute_router_peer.foobar"),
			},
			resource.TestStep{
				Config: testAccComputeRouterPeer_keepRouter,
				Check: testAccCheckComputeRouterPeerDestroy(
					"google_compute_router_peer.foobar"),
			},
		},
	})
}

func testAccCheckComputeRouterPeerDestroy(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)
		project := config.Project

		routersService := compute.NewRoutersService(config.clientCompute)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "google_compute_router_peer" {
				continue
			}

			region := rs.Primary.Attributes["region"]
			name := rs.Primary.Attributes["name"]
			routerName := rs.Primary.Attributes["router"]

			router, err := routersService.Get(project, region, routerName).Do()

			if err != nil {
				return fmt.Errorf("Error Reading Router %s: %s", routerName, err)
			}

			var peerExists bool = false

			var peers []*compute.RouterBgpPeer = router.BgpPeers
			for _, peer := range peers {

				if peer.Name == name {
					peerExists = true
					break
				}
			}

			if peerExists {
				return fmt.Errorf("Peer %s still exists on router %s", name, router.Name)
			}

		}

		return nil
	}
}

func testAccCheckComputeRouterPeerExists(n string) resource.TestCheckFunc {
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
		routerName := rs.Primary.Attributes["router"]
		region := rs.Primary.Attributes["region"]
		project := config.Project

		routersService := compute.NewRoutersService(config.clientCompute)
		router, err := routersService.Get(project, region, routerName).Do()

		if err != nil {
			return fmt.Errorf("Error Reading Router %s: %s", routerName, err)
		}

		var peerExists bool = false

		var peers []*compute.RouterBgpPeer = router.BgpPeers
		for _, peer := range peers {

			if peer.Name == name {
				peerExists = true
				break
			}
		}

		if !peerExists {
			return fmt.Errorf("Peer %s not found for router %s", name, router.Name)
		}

		return nil
	}
}

var testAccComputeRouterPeer_basic = fmt.Sprintf(`
resource "google_compute_network" "foobar" {
        name = "peer-test-%s"
}
resource "google_compute_subnetwork" "foobar" {
        name = "peer-test-%s"
        network = "${google_compute_network.foobar.self_link}"
        ip_cidr_range = "10.0.0.0/16"
        region = "us-central1"
}
resource "google_compute_address" "foobar" {
        name = "peer-test-%s"
        region = "${google_compute_subnetwork.foobar.region}"
}
resource "google_compute_vpn_gateway" "foobar" {
        name = "peer-test-%s"
        network = "${google_compute_network.foobar.self_link}"
        region = "${google_compute_subnetwork.foobar.region}"
}
resource "google_compute_forwarding_rule" "foobar_esp" {
        name = "peer-test-%s"
        region = "${google_compute_vpn_gateway.foobar.region}"
        ip_protocol = "ESP"
        ip_address = "${google_compute_address.foobar.address}"
        target = "${google_compute_vpn_gateway.foobar.self_link}"
}
resource "google_compute_forwarding_rule" "foobar_udp500" {
        name = "peer-test-%s"
        region = "${google_compute_forwarding_rule.foobar_esp.region}"
        ip_protocol = "UDP"
        port_range = "500-500"
        ip_address = "${google_compute_address.foobar.address}"
        target = "${google_compute_vpn_gateway.foobar.self_link}"
}
resource "google_compute_forwarding_rule" "foobar_udp4500" {
        name = "peer-test-%s"
        region = "${google_compute_forwarding_rule.foobar_udp500.region}"
        ip_protocol = "UDP"
        port_range = "4500-4500"
        ip_address = "${google_compute_address.foobar.address}"
        target = "${google_compute_vpn_gateway.foobar.self_link}"
}
resource "google_compute_router" "foobar"{
        name = "peer-test-%s"
        region = "${google_compute_forwarding_rule.foobar_udp500.region}"
        network = "${google_compute_network.foobar.self_link}"
        bgp {
                asn = 64514
        }
}
resource "google_compute_vpn_tunnel" "foobar" {
        name = "peer-test-%s"
        region = "${google_compute_forwarding_rule.foobar_udp4500.region}"
        target_vpn_gateway = "${google_compute_vpn_gateway.foobar.self_link}"
        shared_secret = "unguessable"
        peer_ip = "8.8.8.8"
        router = "${google_compute_router.foobar.name}"
}
resource "google_compute_router_interface" "foobar" {
  name    = "peer-test-%s"
  router  = "${google_compute_router.foobar.name}"
  region  = "${google_compute_router.foobar.region}"
  ip_range = "169.254.3.1/30"
  vpn_tunnel = "${google_compute_vpn_tunnel.foobar.name}"
}
resource "google_compute_router_peer" "foobar" {
  name = "peer-test-%s"
  router  = "${google_compute_router.foobar.name}"
  region  = "${google_compute_router.foobar.region}"
  ip_address = "169.254.3.2"
  asn = 65515
  interface = "${google_compute_router_interface.foobar.name}"
}`, acctest.RandString(10), acctest.RandString(10), acctest.RandString(10),
	acctest.RandString(10), acctest.RandString(10), acctest.RandString(10),
	acctest.RandString(10), acctest.RandString(10), acctest.RandString(10),
	acctest.RandString(10), acctest.RandString(10))

var testAccComputeRouterPeer_keepRouter = fmt.Sprintf(`
resource "google_compute_network" "foobar" {
        name = "peer-test-%s"
}
resource "google_compute_subnetwork" "foobar" {
        name = "peer-test-%s"
        network = "${google_compute_network.foobar.self_link}"
        ip_cidr_range = "10.0.0.0/16"
        region = "us-central1"
}
resource "google_compute_address" "foobar" {
        name = "peer-test-%s"
        region = "${google_compute_subnetwork.foobar.region}"
}
resource "google_compute_vpn_gateway" "foobar" {
        name = "peer-test-%s"
        network = "${google_compute_network.foobar.self_link}"
        region = "${google_compute_subnetwork.foobar.region}"
}
resource "google_compute_forwarding_rule" "foobar_esp" {
        name = "peer-test-%s"
        region = "${google_compute_vpn_gateway.foobar.region}"
        ip_protocol = "ESP"
        ip_address = "${google_compute_address.foobar.address}"
        target = "${google_compute_vpn_gateway.foobar.self_link}"
}
resource "google_compute_forwarding_rule" "foobar_udp500" {
        name = "peer-test-%s"
        region = "${google_compute_forwarding_rule.foobar_esp.region}"
        ip_protocol = "UDP"
        port_range = "500-500"
        ip_address = "${google_compute_address.foobar.address}"
        target = "${google_compute_vpn_gateway.foobar.self_link}"
}
resource "google_compute_forwarding_rule" "foobar_udp4500" {
        name = "peer-test-%s"
        region = "${google_compute_forwarding_rule.foobar_udp500.region}"
        ip_protocol = "UDP"
        port_range = "4500-4500"
        ip_address = "${google_compute_address.foobar.address}"
        target = "${google_compute_vpn_gateway.foobar.self_link}"
}
resource "google_compute_router" "foobar"{
        name = "peer-test-%s"
        region = "${google_compute_forwarding_rule.foobar_udp500.region}"
        network = "${google_compute_network.foobar.self_link}"
        bgp {
                asn = 64514
        }
}
resource "google_compute_vpn_tunnel" "foobar" {
        name = "peer-test-%s"
        region = "${google_compute_forwarding_rule.foobar_udp4500.region}"
        target_vpn_gateway = "${google_compute_vpn_gateway.foobar.self_link}"
        shared_secret = "unguessable"
        peer_ip = "8.8.8.8"
        router = "${google_compute_router.foobar.name}"
}
resource "google_compute_router_interface" "foobar" {
  name    = "peer-test-%s"
  router  = "${google_compute_router.foobar.name}"
  region  = "${google_compute_router.foobar.region}"
  ip_range = "169.254.3.1/30"
  vpn_tunnel = "${google_compute_vpn_tunnel.foobar.name}"
}`, acctest.RandString(10), acctest.RandString(10), acctest.RandString(10),
	acctest.RandString(10), acctest.RandString(10), acctest.RandString(10),
	acctest.RandString(10), acctest.RandString(10), acctest.RandString(10),
	acctest.RandString(10))
