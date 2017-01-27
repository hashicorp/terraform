package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"google.golang.org/api/compute/v1"
)

func TestAccComputeRouter_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeRouterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeRouter_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeRouterExists(
						"google_compute_router.foobar"),
					resource.TestCheckResourceAttr(
						"google_compute_router.foobar", "region", "europe-west1"),
				),
			},
		},
	})
}

func TestAccComputeRouter_noRegion(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeRouterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeRouter_noRegion,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeRouterExists(
						"google_compute_router.foobar"),
					resource.TestCheckResourceAttr(
						"google_compute_router.foobar", "region", "us-central1"),
				),
			},
		},
	})
}

func TestAccComputeRouter_networkLink(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeRouterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeRouter_networkLink,
				Check: testAccCheckComputeRouterExists(
					"google_compute_router.foobar"),
			},
		},
	})
}

func testAccCheckComputeRouterDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	project := config.Project

	routersService := compute.NewRoutersService(config.clientCompute)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_router" {
			continue
		}

		region := rs.Primary.Attributes["region"]
		name := rs.Primary.Attributes["name"]

		_, err := routersService.Get(project, region, name).Do()

		if err == nil {
			return fmt.Errorf("Error, Router %s in region %s still exists",
				name, region)
		}
	}

	return nil
}

func testAccCheckComputeRouterExists(n string) resource.TestCheckFunc {
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

		routersService := compute.NewRoutersService(config.clientCompute)
		_, err := routersService.Get(project, region, name).Do()

		if err != nil {
			return fmt.Errorf("Error Reading Router %s: %s", name, err)
		}

		return nil
	}
}

var testAccComputeRouter_basic = fmt.Sprintf(`
resource "google_compute_network" "foobar" {
	name = "router-test-%s"
}
resource "google_compute_subnetwork" "foobar" {
	name = "router-test-%s"
	network = "${google_compute_network.foobar.self_link}"
	ip_cidr_range = "10.0.0.0/16"
	region = "europe-west1"
}
resource "google_compute_router" "foobar" {
        name = "router-test-%s"
	region = "${google_compute_subnetwork.foobar.region}"
	network = "${google_compute_network.foobar.name}"
        bgp { 
           asn = 64514
       }
}
`, acctest.RandString(10), acctest.RandString(10), acctest.RandString(10))

var testAccComputeRouter_noRegion = fmt.Sprintf(`
resource "google_compute_network" "foobar" {
	name = "router-test-%s"
}
resource "google_compute_subnetwork" "foobar" {
	name = "router-test-%s"
	network = "${google_compute_network.foobar.self_link}"
	ip_cidr_range = "10.0.0.0/16"
	region = "us-central1"
}
resource "google_compute_router" "foobar" {
        name = "router-test-%s"
	network = "${google_compute_network.foobar.name}"
        bgp {
           asn = 64514
       }
}
`, acctest.RandString(10), acctest.RandString(10), acctest.RandString(10))

var testAccComputeRouter_networkLink = fmt.Sprintf(`
resource "google_compute_network" "foobar" {
	name = "router-test-%s"
}
resource "google_compute_subnetwork" "foobar" {
	name = "router-test-%s"
	network = "${google_compute_network.foobar.self_link}"
	ip_cidr_range = "10.0.0.0/16"
	region = "us-central1"
}
resource "google_compute_router" "foobar" {
        name = "router-test-%s"
	region = "${google_compute_subnetwork.foobar.region}"
	network = "${google_compute_network.foobar.self_link}"
        bgp {
           asn = 64514
       }
}
`, acctest.RandString(10), acctest.RandString(10), acctest.RandString(10))
