package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccComputeRouter_basic(t *testing.T) {
	resourceRegion := "europe-west1"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeRouterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeRouterBasic(resourceRegion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeRouterExists(
						"google_compute_router.foobar"),
					resource.TestCheckResourceAttr(
						"google_compute_router.foobar", "region", resourceRegion),
				),
			},
		},
	})
}

func TestAccComputeRouter_noRegion(t *testing.T) {
	providerRegion := "us-central1"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeRouterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeRouterNoRegion(providerRegion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeRouterExists(
						"google_compute_router.foobar"),
					resource.TestCheckResourceAttr(
						"google_compute_router.foobar", "region", providerRegion),
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
				Config: testAccComputeRouterNetworkLink(),
				Check: testAccCheckComputeRouterExists(
					"google_compute_router.foobar"),
			},
		},
	})
}

func testAccCheckComputeRouterDestroy(s *terraform.State) error {
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

		name := rs.Primary.Attributes["name"]

		_, err = routersService.Get(project, region, name).Do()

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

		project, err := getTestProject(rs.Primary, config)
		if err != nil {
			return err
		}

		region, err := getTestRegion(rs.Primary, config)
		if err != nil {
			return err
		}

		name := rs.Primary.Attributes["name"]

		routersService := config.clientCompute.Routers
		_, err = routersService.Get(project, region, name).Do()

		if err != nil {
			return fmt.Errorf("Error Reading Router %s: %s", name, err)
		}

		return nil
	}
}

func testAccComputeRouterBasic(resourceRegion string) string {
	testId := acctest.RandString(10)
	return fmt.Sprintf(`
		resource "google_compute_network" "foobar" {
			name = "router-test-%s"
		}
		resource "google_compute_subnetwork" "foobar" {
			name = "router-test-%s"
			network = "${google_compute_network.foobar.self_link}"
			ip_cidr_range = "10.0.0.0/16"
			region = "%s"
		}
		resource "google_compute_router" "foobar" {
			name = "router-test-%s"
			region = "${google_compute_subnetwork.foobar.region}"
			network = "${google_compute_network.foobar.name}"
			bgp {
				asn = 64514
			}
		}
	`, testId, testId, resourceRegion, testId)
}

func testAccComputeRouterNoRegion(providerRegion string) string {
	testId := acctest.RandString(10)
	return fmt.Sprintf(`
		resource "google_compute_network" "foobar" {
			name = "router-test-%s"
		}
		resource "google_compute_subnetwork" "foobar" {
			name = "router-test-%s"
			network = "${google_compute_network.foobar.self_link}"
			ip_cidr_range = "10.0.0.0/16"
			region = "%s"
		}
		resource "google_compute_router" "foobar" {
			name = "router-test-%s"
			network = "${google_compute_network.foobar.name}"
			bgp {
				asn = 64514
			}
		}
	`, testId, testId, providerRegion, testId)
}

func testAccComputeRouterNetworkLink() string {
	testId := acctest.RandString(10)
	return fmt.Sprintf(`
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
			network = "${google_compute_network.foobar.self_link}"
			bgp {
				asn = 64514
			}
		}
	`, testId, testId, testId)
}
