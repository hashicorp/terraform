package google

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccContainerNodePool_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckContainerNodePoolDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerNodePool_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContainerNodePoolMatches("google_container_node_pool.np"),
				),
			},
		},
	})
}

func testAccCheckContainerNodePoolDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_container_node_pool" {
			continue
		}

		attributes := rs.Primary.Attributes
		_, err := config.clientContainer.Projects.Zones.Clusters.NodePools.Get(
			config.Project, attributes["zone"], attributes["cluster"], attributes["name"]).Do()
		if err == nil {
			return fmt.Errorf("NodePool still exists")
		}
	}

	return nil
}

func testAccCheckContainerNodePoolMatches(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)

		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		attributes := rs.Primary.Attributes
		found, err := config.clientContainer.Projects.Zones.Clusters.NodePools.Get(
			config.Project, attributes["zone"], attributes["cluster"], attributes["name"]).Do()
		if err != nil {
			return err
		}

		if found.Name != attributes["name"] {
			return fmt.Errorf("NodePool not found")
		}

		inc, err := strconv.Atoi(attributes["initial_node_count"])
		if err != nil {
			return err
		}
		if found.InitialNodeCount != int64(inc) {
			return fmt.Errorf("Mismatched initialNodeCount. TF State: %s. GCP State: %d",
				attributes["initial_node_count"], found.InitialNodeCount)
		}
		return nil
	}
}

var testAccContainerNodePool_basic = fmt.Sprintf(`
resource "google_container_cluster" "cluster" {
	name = "tf-cluster-nodepool-test-%s"
	zone = "us-central1-a"
	initial_node_count = 3

	master_auth {
		username = "mr.yoda"
		password = "adoy.rm"
	}
}

resource "google_container_node_pool" "np" {
	name = "tf-nodepool-test-%s"
	zone = "us-central1-a"
	cluster = "${google_container_cluster.cluster.name}"
	initial_node_count = 2
}`, acctest.RandString(10), acctest.RandString(10))
