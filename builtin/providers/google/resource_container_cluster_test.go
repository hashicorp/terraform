package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccContainerCluster_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckContainerClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerCluster_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContainerClusterExists(
						"google_container_cluster.primary"),
				),
			},
		},
	})
}

func TestAccContainerCluster_withNodeConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckContainerClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerCluster_withNodeConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContainerClusterExists(
						"google_container_cluster.with_node_config"),
				),
			},
		},
	})
}

func testAccCheckContainerClusterDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_container_cluster" {
			continue
		}

		attributes := rs.Primary.Attributes
		_, err := config.clientContainer.Projects.Zones.Clusters.Get(
			config.Project, attributes["zone"], attributes["name"]).Do()
		if err == nil {
			return fmt.Errorf("Cluster still exists")
		}
	}

	return nil
}

func testAccCheckContainerClusterExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		attributes := rs.Primary.Attributes
		found, err := config.clientContainer.Projects.Zones.Clusters.Get(
			config.Project, attributes["zone"], attributes["name"]).Do()
		if err != nil {
			return err
		}

		if found.Name != attributes["name"] {
			return fmt.Errorf("Cluster not found")
		}

		return nil
	}
}

var testAccContainerCluster_basic = fmt.Sprintf(`
resource "google_container_cluster" "primary" {
	name = "cluster-test-%s"
	zone = "us-central1-a"
	initial_node_count = 3

	master_auth {
		username = "mr.yoda"
		password = "adoy.rm"
	}
}`, acctest.RandString(10))

var testAccContainerCluster_withNodeConfig = fmt.Sprintf(`
resource "google_container_cluster" "with_node_config" {
	name = "cluster-test-%s"
	zone = "us-central1-f"
	initial_node_count = 1

	master_auth {
		username = "mr.yoda"
		password = "adoy.rm"
	}

	node_config {
		machine_type = "g1-small"
		disk_size_gb = 15
		oauth_scopes = [
			"https://www.googleapis.com/auth/compute",
			"https://www.googleapis.com/auth/devstorage.read_only",
			"https://www.googleapis.com/auth/logging.write",
			"https://www.googleapis.com/auth/monitoring"
		]
	}
}`, acctest.RandString(10))
