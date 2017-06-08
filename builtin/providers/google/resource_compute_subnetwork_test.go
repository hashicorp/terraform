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
	var subnetwork1 compute.Subnetwork
	var subnetwork2 compute.Subnetwork

	cnName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	subnetwork1Name := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	subnetwork2Name := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	subnetwork3Name := fmt.Sprintf("tf-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeSubnetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeSubnetwork_basic(cnName, subnetwork1Name, subnetwork2Name, subnetwork3Name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeSubnetworkExists(
						"google_compute_subnetwork.network-ref-by-url", &subnetwork1),
					testAccCheckComputeSubnetworkExists(
						"google_compute_subnetwork.network-ref-by-name", &subnetwork2),
				),
			},
		},
	})
}

func TestAccComputeSubnetwork_update(t *testing.T) {
	var subnetwork compute.Subnetwork

	cnName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	subnetworkName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeSubnetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeSubnetwork_update1(cnName, subnetworkName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeSubnetworkExists(
						"google_compute_subnetwork.network-with-private-google-access", &subnetwork),
				),
			},
			resource.TestStep{
				Config: testAccComputeSubnetwork_update2(cnName, subnetworkName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeSubnetworkExists(
						"google_compute_subnetwork.network-with-private-google-access", &subnetwork),
				),
			},
		},
	})

	if subnetwork.PrivateIpGoogleAccess {
		t.Errorf("Expected PrivateIpGoogleAccess to be false, got %v", subnetwork.PrivateIpGoogleAccess)
	}
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

func testAccComputeSubnetwork_basic(cnName, subnetwork1Name, subnetwork2Name, subnetwork3Name string) string {
	return fmt.Sprintf(`
resource "google_compute_network" "custom-test" {
	name = "%s"
	auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "network-ref-by-url" {
	name = "%s"
	ip_cidr_range = "10.0.0.0/16"
	region = "us-central1"
	network = "${google_compute_network.custom-test.self_link}"
}


resource "google_compute_subnetwork" "network-ref-by-name" {
	name = "%s"
	ip_cidr_range = "10.1.0.0/16"
	region = "us-central1"
	network = "${google_compute_network.custom-test.name}"
}

resource "google_compute_subnetwork" "network-with-private-google-access" {
	name = "%s"
	ip_cidr_range = "10.2.0.0/16"
	region = "us-central1"
	network = "${google_compute_network.custom-test.self_link}"
	private_ip_google_access = true
}
`, cnName, subnetwork1Name, subnetwork2Name, subnetwork3Name)
}

func testAccComputeSubnetwork_update1(cnName, subnetworkName string) string {
	return fmt.Sprintf(`
resource "google_compute_network" "custom-test" {
	name = "%s"
	auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "network-with-private-google-access" {
	name = "%s"
	ip_cidr_range = "10.2.0.0/16"
	region = "us-central1"
	network = "${google_compute_network.custom-test.self_link}"
	private_ip_google_access = true
}
`, cnName, subnetworkName)
}

func testAccComputeSubnetwork_update2(cnName, subnetworkName string) string {
	return fmt.Sprintf(`
resource "google_compute_network" "custom-test" {
	name = "%s"
	auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "network-with-private-google-access" {
	name = "%s"
	ip_cidr_range = "10.2.0.0/16"
	region = "us-central1"
	network = "${google_compute_network.custom-test.self_link}"
}
`, cnName, subnetworkName)
}
