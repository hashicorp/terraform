package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccComputeRouterPeer_basic(t *testing.T) {
	testId := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeRouterPeerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeRouterPeerBasic(testId),
				Check: testAccCheckComputeRouterPeerExists(
					"google_compute_router_peer.foobar"),
			},
			resource.TestStep{
				Config: testAccComputeRouterPeerKeepRouter(testId),
				Check: testAccCheckComputeRouterPeerDelete(
					"google_compute_router_peer.foobar"),
			},
		},
	})
}

func testAccCheckComputeRouterPeerDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	routersService := config.clientCompute.Routers

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_router" {
			continue
		}

		project, err := getTestProject(rs.Primary, config)
		if err != nil {
			return err
		}

		region, err := getTestRegion(rs.Primary, config)
		if err != nil {
			return err
		}

		routerName := rs.Primary.Attributes["router"]

		_, err = routersService.Get(project, region, routerName).Do()

		if err == nil {
			return fmt.Errorf("Error, Router %s in region %s still exists",
				routerName, region)
		}
	}

	return nil
}

func testAccCheckComputeRouterPeerDelete(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)

		routersService := config.clientCompute.Routers

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "google_compute_router_peer" {
				continue
			}

			project, err := getTestProject(rs.Primary, config)
			if err != nil {
				return err
			}

			region, err := getTestRegion(rs.Primary, config)
			if err != nil {
				return err
			}

			name := rs.Primary.Attributes["name"]
			routerName := rs.Primary.Attributes["router"]

			router, err := routersService.Get(project, region, routerName).Do()

			if err != nil {
				return fmt.Errorf("Error Reading Router %s: %s", routerName, err)
			}

			peers := router.BgpPeers
			for _, peer := range peers {

				if peer.Name == name {
					return fmt.Errorf("Peer %s still exists on router %s/%s", name, region, router.Name)
				}
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

		project, err := getTestProject(rs.Primary, config)
		if err != nil {
			return err
		}

		region, err := getTestRegion(rs.Primary, config)
		if err != nil {
			return err
		}

		name := rs.Primary.Attributes["name"]
		routerName := rs.Primary.Attributes["router"]

		routersService := config.clientCompute.Routers
		router, err := routersService.Get(project, region, routerName).Do()

		if err != nil {
			return fmt.Errorf("Error Reading Router %s: %s", routerName, err)
		}

		for _, peer := range router.BgpPeers {

			if peer.Name == name {
				return nil
			}
		}

		return fmt.Errorf("Peer %s not found for router %s", name, router.Name)
	}
}

func testAccComputeRouterPeerBasic(testId string) string {
	return fmt.Sprintf(`
	        resource "google_compute_network" "foobar" {
			name = "router-peer-test-%s"
		}
		resource "google_compute_subnetwork" "foobar" {
			name = "router-peer-test-%s"
			network = "${google_compute_network.foobar.self_link}"
			ip_cidr_range = "10.0.0.0/16"
			region = "us-central1"
		}
		resource "google_compute_address" "foobar" {
			name = "router-peer-test-%s"
			region = "${google_compute_subnetwork.foobar.region}"
		}
		resource "google_compute_vpn_gateway" "foobar" {
			name = "router-peer-test-%s"
			network = "${google_compute_network.foobar.self_link}"
			region = "${google_compute_subnetwork.foobar.region}"
		}
		resource "google_compute_forwarding_rule" "foobar_esp" {
			name = "router-peer-test-%s-1"
			region = "${google_compute_vpn_gateway.foobar.region}"
			ip_protocol = "ESP"
			ip_address = "${google_compute_address.foobar.address}"
			target = "${google_compute_vpn_gateway.foobar.self_link}"
		}
		resource "google_compute_forwarding_rule" "foobar_udp500" {
			name = "router-peer-test-%s-2"
			region = "${google_compute_forwarding_rule.foobar_esp.region}"
			ip_protocol = "UDP"
			port_range = "500-500"
			ip_address = "${google_compute_address.foobar.address}"
			target = "${google_compute_vpn_gateway.foobar.self_link}"
		}
		resource "google_compute_forwarding_rule" "foobar_udp4500" {
			name = "router-peer-test-%s-3"
			region = "${google_compute_forwarding_rule.foobar_udp500.region}"
			ip_protocol = "UDP"
			port_range = "4500-4500"
			ip_address = "${google_compute_address.foobar.address}"
			target = "${google_compute_vpn_gateway.foobar.self_link}"
		}
		resource "google_compute_router" "foobar"{
			name = "router-peer-test-%s"
			region = "${google_compute_forwarding_rule.foobar_udp500.region}"
			network = "${google_compute_network.foobar.self_link}"
			bgp {
				asn = 64514
			}
		}
		resource "google_compute_vpn_tunnel" "foobar" {
			name = "router-peer-test-%s"
			region = "${google_compute_forwarding_rule.foobar_udp4500.region}"
			target_vpn_gateway = "${google_compute_vpn_gateway.foobar.self_link}"
			shared_secret = "unguessable"
			peer_ip = "8.8.8.8"
			router = "${google_compute_router.foobar.name}"
		}
		resource "google_compute_router_interface" "foobar" {
			name = "router-peer-test-%s"
			router = "${google_compute_router.foobar.name}"
			region = "${google_compute_router.foobar.region}"
			ip_range = "169.254.3.1/30"
			vpn_tunnel = "${google_compute_vpn_tunnel.foobar.name}"
		}
		resource "google_compute_router_peer" "foobar" {
			name = "router-peer-test-%s"
			router = "${google_compute_router.foobar.name}"
			region = "${google_compute_router.foobar.region}"
			peer_ip_address = "169.254.3.2"
			peer_asn = 65515
			advertised_route_priority = 100
			interface = "${google_compute_router_interface.foobar.name}"
		}
	`, testId, testId, testId, testId, testId, testId, testId, testId, testId, testId, testId)
}

func testAccComputeRouterPeerKeepRouter(testId string) string {
	return fmt.Sprintf(`
		resource "google_compute_network" "foobar" {
			name = "router-peer-test-%s"
		}
		resource "google_compute_subnetwork" "foobar" {
			name = "router-peer-test-%s"
			network = "${google_compute_network.foobar.self_link}"
			ip_cidr_range = "10.0.0.0/16"
			region = "us-central1"
		}
		resource "google_compute_address" "foobar" {
			name = "router-peer-test-%s"
			region = "${google_compute_subnetwork.foobar.region}"
		}
		resource "google_compute_vpn_gateway" "foobar" {
			name = "router-peer-test-%s"
			network = "${google_compute_network.foobar.self_link}"
			region = "${google_compute_subnetwork.foobar.region}"
		}
		resource "google_compute_forwarding_rule" "foobar_esp" {
			name = "router-peer-test-%s-1"
			region = "${google_compute_vpn_gateway.foobar.region}"
			ip_protocol = "ESP"
			ip_address = "${google_compute_address.foobar.address}"
			target = "${google_compute_vpn_gateway.foobar.self_link}"
		}
		resource "google_compute_forwarding_rule" "foobar_udp500" {
			name = "router-peer-test-%s-2"
			region = "${google_compute_forwarding_rule.foobar_esp.region}"
			ip_protocol = "UDP"
			port_range = "500-500"
			ip_address = "${google_compute_address.foobar.address}"
			target = "${google_compute_vpn_gateway.foobar.self_link}"
		}
		resource "google_compute_forwarding_rule" "foobar_udp4500" {
			name = "router-peer-test-%s-3"
			region = "${google_compute_forwarding_rule.foobar_udp500.region}"
			ip_protocol = "UDP"
			port_range = "4500-4500"
			ip_address = "${google_compute_address.foobar.address}"
			target = "${google_compute_vpn_gateway.foobar.self_link}"
		}
		resource "google_compute_router" "foobar"{
			name = "router-peer-test-%s"
			region = "${google_compute_forwarding_rule.foobar_udp500.region}"
			network = "${google_compute_network.foobar.self_link}"
			bgp {
				asn = 64514
			}
		}
		resource "google_compute_vpn_tunnel" "foobar" {
			name = "router-peer-test-%s"
			region = "${google_compute_forwarding_rule.foobar_udp4500.region}"
			target_vpn_gateway = "${google_compute_vpn_gateway.foobar.self_link}"
			shared_secret = "unguessable"
			peer_ip = "8.8.8.8"
			router = "${google_compute_router.foobar.name}"
		}
		resource "google_compute_router_interface" "foobar" {
			name = "router-peer-test-%s"
			router = "${google_compute_router.foobar.name}"
			region = "${google_compute_router.foobar.region}"
			ip_range = "169.254.3.1/30"
			vpn_tunnel = "${google_compute_vpn_tunnel.foobar.name}"
		}
	`, testId, testId, testId, testId, testId, testId, testId, testId, testId, testId)
}
