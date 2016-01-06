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

var testAccComputeNetwork_basic = fmt.Sprintf(`
resource "google_compute_network" "foobar" {
	name = "network-test-%s"
	ipv4_range = "10.0.0.0/16"
}`, acctest.RandString(10))
