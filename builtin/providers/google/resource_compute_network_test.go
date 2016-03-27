package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/compute/v1"
)

func TestAccComputeNetwork_basic(t *testing.T) {
	var network compute.Network

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeNetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeNetwork_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeNetworkExists(
						"google_compute_network.foobar", &network),
				),
			},
		},
	})
}

func TestAccComputeNetwork_auto_subnet(t *testing.T) {
	var network compute.Network

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeNetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeNetwork_auto_subnet,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeNetworkExists(
						"google_compute_network.bar", &network),
					testAccCheckComputeNetworkIsAutoSubnet(
						"google_compute_network.bar", &network),
				),
			},
		},
	})
}

func TestAccComputeNetwork_custom_subnet(t *testing.T) {
	var network compute.Network

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeNetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeNetwork_custom_subnet,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeNetworkExists(
						"google_compute_network.baz", &network),
					testAccCheckComputeNetworkIsCustomSubnet(
						"google_compute_network.baz", &network),
				),
			},
		},
	})
}

func testAccCheckComputeNetworkDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_network" {
			continue
		}

		_, err := config.clientCompute.Networks.Get(
			config.Project, rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("Network still exists")
		}
	}

	return nil
}

func testAccCheckComputeNetworkExists(n string, network *compute.Network) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.Networks.Get(
			config.Project, rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("Network not found")
		}

		*network = *found

		return nil
	}
}

func testAccCheckComputeNetworkIsAutoSubnet(n string, network *compute.Network) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.Networks.Get(
			config.Project, network.Name).Do()
		if err != nil {
			return err
		}

		if !found.AutoCreateSubnetworks {
			return fmt.Errorf("should have AutoCreateSubnetworks = true")
		}

		if found.IPv4Range != "" {
			return fmt.Errorf("should not have IPv4Range")
		}

		return nil
	}
}

func testAccCheckComputeNetworkIsCustomSubnet(n string, network *compute.Network) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.Networks.Get(
			config.Project, network.Name).Do()
		if err != nil {
			return err
		}

		if found.AutoCreateSubnetworks {
			return fmt.Errorf("should have AutoCreateSubnetworks = false")
		}

		if found.IPv4Range != "" {
			return fmt.Errorf("should not have IPv4Range")
		}

		return nil
	}
}

var testAccComputeNetwork_basic = fmt.Sprintf(`
resource "google_compute_network" "foobar" {
	name = "network-test-%s"
	ipv4_range = "10.0.0.0/16"
}`, acctest.RandString(10))

var testAccComputeNetwork_auto_subnet = fmt.Sprintf(`
resource "google_compute_network" "bar" {
	name = "network-test-%s"
	auto_create_subnetworks = true
}`, acctest.RandString(10))

var testAccComputeNetwork_custom_subnet = fmt.Sprintf(`
resource "google_compute_network" "baz" {
	name = "network-test-%s"
	auto_create_subnetworks = false
}`, acctest.RandString(10))
