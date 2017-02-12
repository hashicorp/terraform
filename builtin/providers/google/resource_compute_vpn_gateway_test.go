package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"google.golang.org/api/compute/v1"
)

func TestAccComputeVpnGateway_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeVpnGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeVpnGateway_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeVpnGatewayExists(
						"google_compute_vpn_gateway.foobar"),
					testAccCheckComputeVpnGatewayExists(
						"google_compute_vpn_gateway.baz"),
				),
			},
		},
	})
}

func testAccCheckComputeVpnGatewayDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	project := config.Project

	vpnGatewaysService := compute.NewTargetVpnGatewaysService(config.clientCompute)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_network" {
			continue
		}

		region := rs.Primary.Attributes["region"]
		name := rs.Primary.Attributes["name"]

		_, err := vpnGatewaysService.Get(project, region, name).Do()

		if err == nil {
			return fmt.Errorf("Error, VPN Gateway %s in region %s still exists",
				name, region)
		}
	}

	return nil
}

func testAccCheckComputeVpnGatewayExists(n string) resource.TestCheckFunc {
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

		vpnGatewaysService := compute.NewTargetVpnGatewaysService(config.clientCompute)
		_, err := vpnGatewaysService.Get(project, region, name).Do()

		if err != nil {
			return fmt.Errorf("Error Reading VPN Gateway %s: %s", name, err)
		}

		return nil
	}
}

var testAccComputeVpnGateway_basic = fmt.Sprintf(`
resource "google_compute_network" "foobar" {
	name = "gateway-test-%s"
	ipv4_range = "10.0.0.0/16"
}
resource "google_compute_vpn_gateway" "foobar" {
	name = "gateway-test-%s"
	network = "${google_compute_network.foobar.self_link}"
	region = "us-central1"
}
resource "google_compute_vpn_gateway" "baz" {
	name = "gateway-test-%s"
	network = "${google_compute_network.foobar.name}"
	region = "us-central1"
}`, acctest.RandString(10), acctest.RandString(10), acctest.RandString(10))
