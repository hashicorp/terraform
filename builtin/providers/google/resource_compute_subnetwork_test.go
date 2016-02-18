package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/compute/v1"
)

func TestAccComputeSubnetwork_basic(t *testing.T) {
	var subnetwork compute.Subnetwork

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeSubnetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeSubnetwork_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeSubnetworkExists(
						"google_compute_subnetwork.foobar", &subnetwork),
				),
			},
		},
	})
}

func testAccCheckComputeSubnetworkDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_subnetwork" {
			continue
		}

		region, subnet_name := splitSubnetID(rs.Primary.ID)
		_, err := config.clientCompute.Subnetworks.Get(
			config.Project, region, subnet_name).Do()
		if err == nil {
			return fmt.Errorf("Network still exists")
		}
	}

	return nil
}

func testAccCheckComputeSubnetworkExists(n string, subnetwork *compute.Subnetwork) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		region, subnet_name := splitSubnetID(rs.Primary.ID)
		found, err := config.clientCompute.Subnetworks.Get(
			config.Project, region, subnet_name).Do()
		if err != nil {
			return err
		}

		if found.Name != subnet_name {
			return fmt.Errorf("Subnetwork not found")
		}

		*subnetwork = *found

		return nil
	}
}

var testAccComputeSubnetwork_basic = fmt.Sprintf(`
resource "google_compute_network" "custom-test" {
	name = "network-test-%s"
	auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "foobar" {
	name = "subnetwork-test-%s"
	ip_cidr_range = "10.0.0.0/16"
	region = "us-central1"
	network = "${google_compute_network.custom-test.self_link}"
}`, acctest.RandString(10), acctest.RandString(10))
