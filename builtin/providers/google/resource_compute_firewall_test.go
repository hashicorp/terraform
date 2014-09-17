package google

import (
	"fmt"
	"testing"

	"code.google.com/p/google-api-go-client/compute/v1"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccComputeFirewall_basic(t *testing.T) {
	var firewall compute.Firewall

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeFirewallDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeFirewall_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeFirewallExists(
						"google_compute_firewall.foobar", &firewall),
				),
			},
		},
	})
}

func TestAccComputeFirewall_update(t *testing.T) {
	var firewall compute.Firewall

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeFirewallDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeFirewall_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeFirewallExists(
						"google_compute_firewall.foobar", &firewall),
				),
			},
			resource.TestStep{
				Config: testAccComputeFirewall_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeFirewallExists(
						"google_compute_firewall.foobar", &firewall),
					testAccCheckComputeFirewallPorts(
						&firewall, "80-255"),
				),
			},
		},
	})
}

func testAccCheckComputeFirewallDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_firewall" {
			continue
		}

		_, err := config.clientCompute.Firewalls.Get(
			config.Project, rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("Firewall still exists")
		}
	}

	return nil
}

func testAccCheckComputeFirewallExists(n string, firewall *compute.Firewall) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.Firewalls.Get(
			config.Project, rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("Firewall not found")
		}

		*firewall = *found

		return nil
	}
}

func testAccCheckComputeFirewallPorts(
	firewall *compute.Firewall, ports string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(firewall.Allowed) == 0 {
			return fmt.Errorf("no allowed rules")
		}

		if firewall.Allowed[0].Ports[0] != ports {
			return fmt.Errorf("bad: %#v", firewall.Allowed[0].Ports)
		}

		return nil
	}
}

const testAccComputeFirewall_basic = `
resource "google_compute_network" "foobar" {
	name = "terraform-test"
	ipv4_range = "10.0.0.0/16"
}

resource "google_compute_firewall" "foobar" {
	name = "terraform-test"
	network = "${google_compute_network.foobar.name}"
	source_tags = ["foo"]

	allow {
		protocol = "icmp"
	}
}`

const testAccComputeFirewall_update = `
resource "google_compute_network" "foobar" {
	name = "terraform-test"
	ipv4_range = "10.0.0.0/16"
}

resource "google_compute_firewall" "foobar" {
	name = "terraform-test"
	network = "${google_compute_network.foobar.name}"
	source_tags = ["foo"]

	allow {
		protocol = "tcp"
		ports = ["80-255"]
	}
}`
