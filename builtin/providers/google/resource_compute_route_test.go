package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/compute/v1"
)

func TestAccComputeRoute_basic(t *testing.T) {
	var route compute.Route

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeRouteDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeRoute_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeRouteExists(
						"google_compute_route.foobar", &route),
				),
			},
		},
	})
}

func TestAccComputeRoute_defaultInternetGateway(t *testing.T) {
	var route compute.Route

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeRouteDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeRoute_defaultInternetGateway,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeRouteExists(
						"google_compute_route.foobar", &route),
				),
			},
		},
	})
}

func testAccCheckComputeRouteDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_route" {
			continue
		}

		_, err := config.clientCompute.Routes.Get(
			config.Project, rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("Route still exists")
		}
	}

	return nil
}

func testAccCheckComputeRouteExists(n string, route *compute.Route) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.Routes.Get(
			config.Project, rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("Route not found")
		}

		*route = *found

		return nil
	}
}

var testAccComputeRoute_basic = fmt.Sprintf(`
resource "google_compute_network" "foobar" {
	name = "route-test-%s"
	ipv4_range = "10.0.0.0/16"
}

resource "google_compute_route" "foobar" {
	name = "route-test-%s"
	dest_range = "15.0.0.0/24"
	network = "${google_compute_network.foobar.name}"
	next_hop_ip = "10.0.1.5"
	priority = 100
}`, acctest.RandString(10), acctest.RandString(10))

var testAccComputeRoute_defaultInternetGateway = fmt.Sprintf(`
resource "google_compute_network" "foobar" {
	name = "route-test-%s"
	ipv4_range = "10.0.0.0/16"
}

resource "google_compute_route" "foobar" {
	name = "route-test-%s"
	dest_range = "0.0.0.0/0"
	network = "${google_compute_network.foobar.name}"
	next_hop_gateway = "default-internet-gateway"
	priority = 100
}`, acctest.RandString(10), acctest.RandString(10))
