package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"strconv"
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

func TestAccContainerCluster_withAdditionalZones(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckContainerClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerCluster_withAdditionalZones,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContainerClusterExists(
						"google_container_cluster.with_additional_zones"),
					testAccCheckContainerClusterAdditionalZonesExist(
						"google_container_cluster.with_additional_zones", 2),
				),
			},
		},
	})
}

func TestAccContainerCluster_withVersion(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckContainerClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerCluster_withVersion,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContainerClusterExists(
						"google_container_cluster.with_version"),
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

func TestAccContainerCluster_withNodeConfigScopeAlias(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckContainerClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerCluster_withNodeConfigScopeAlias,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContainerClusterExists(
						"google_container_cluster.with_node_config_scope_alias"),
				),
			},
		},
	})
}

func TestAccContainerCluster_network(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckContainerClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerCluster_networkRef,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContainerClusterExists(
						"google_container_cluster.with_net_ref_by_url"),
					testAccCheckContainerClusterExists(
						"google_container_cluster.with_net_ref_by_name"),
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

func testAccCheckContainerClusterAdditionalZonesExist(n string, num int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		additionalZonesSize, err := strconv.Atoi(rs.Primary.Attributes["additional_zones.#"])
		if err != nil {
			return err
		}
		if additionalZonesSize != num {
			return fmt.Errorf("number of additional zones did not match %d, was %d", num, additionalZonesSize)
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

var testAccContainerCluster_withAdditionalZones = fmt.Sprintf(`
resource "google_container_cluster" "with_additional_zones" {
	name = "cluster-test-%s"
	zone = "us-central1-a"
	initial_node_count = 1

	additional_zones = [
		"us-central1-b",
		"us-central1-c"
	]

	master_auth {
		username = "mr.yoda"
		password = "adoy.rm"
	}
}`, acctest.RandString(10))

var testAccContainerCluster_withVersion = fmt.Sprintf(`
resource "google_container_cluster" "with_version" {
	name = "cluster-test-%s"
	zone = "us-central1-a"
	node_version = "1.5.2"
	initial_node_count = 1

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

var testAccContainerCluster_withNodeConfigScopeAlias = fmt.Sprintf(`
resource "google_container_cluster" "with_node_config_scope_alias" {
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
		oauth_scopes = [ "compute-rw", "storage-ro", "logging-write", "monitoring" ]
	}
}`, acctest.RandString(10))

var testAccContainerCluster_networkRef = fmt.Sprintf(`
resource "google_compute_network" "container_network" {
	name = "container-net-%s"
	auto_create_subnetworks = true
}

resource "google_container_cluster" "with_net_ref_by_url" {
	name = "cluster-test-%s"
	zone = "us-central1-a"
	initial_node_count = 1

	master_auth {
		username = "mr.yoda"
		password = "adoy.rm"
	}

	network = "${google_compute_network.container_network.self_link}"
}

resource "google_container_cluster" "with_net_ref_by_name" {
	name = "cluster-test-%s"
	zone = "us-central1-a"
	initial_node_count = 1

	master_auth {
		username = "mr.yoda"
		password = "adoy.rm"
	}

	network = "${google_compute_network.container_network.name}"
}`, acctest.RandString(10), acctest.RandString(10), acctest.RandString(10))
