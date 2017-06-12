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

func TestAccContainerNodePool_withNodeConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckContainerNodePoolDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerNodePool_withNodeConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContainerNodePoolMatches("google_container_node_pool.np_with_node_config"),
				),
			},
		},
	})
}

func TestAccContainerNodePool_withNodeConfigScopeAlias(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckContainerNodePoolDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerNodePool_withNodeConfigScopeAlias,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContainerNodePoolMatches("google_container_node_pool.np_with_node_config_scope_alias"),
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
		nodepool, err := config.clientContainer.Projects.Zones.Clusters.NodePools.Get(
			config.Project, attributes["zone"], attributes["cluster"], attributes["name"]).Do()
		if err != nil {
			return err
		}

		if nodepool.Name != attributes["name"] {
			return fmt.Errorf("NodePool not found")
		}

		type nodepoolTestField struct {
			tfAttr  string
			gcpAttr interface{}
		}

		nodepoolTests := []nodepoolTestField{
			{"initial_node_count", strconv.FormatInt(nodepool.InitialNodeCount, 10)},
			{"node_config.0.machine_type", nodepool.Config.MachineType},
			{"node_config.0.disk_size_gb", strconv.FormatInt(nodepool.Config.DiskSizeGb, 10)},
			{"node_config.0.oauth_scopes", nodepool.Config.OauthScopes},
		}

		for _, attrs := range nodepoolTests {
			if c := nodepoolCheckMatch(attributes, attrs.tfAttr, attrs.gcpAttr); c != "" {
				return fmt.Errorf(c)
			}
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

var testAccContainerNodePool_withNodeConfig = fmt.Sprintf(`
resource "google_container_cluster" "cluster" {
	name = "tf-cluster-nodepool-test-%s"
	zone = "us-central1-a"
	initial_node_count = 1

	master_auth {
		username = "mr.yoda"
		password = "adoy.rm"
	}
}

resource "google_container_node_pool" "np_with_node_config" {
	name = "tf-nodepool-test-%s"
	zone = "us-central1-a"
	cluster = "${google_container_cluster.cluster.name}"
	initial_node_count = 1

	node_config {
		machine_type = "g1-small"
		disk_size_gb = 10
		oauth_scopes = [
			"https://www.googleapis.com/auth/compute",
			"https://www.googleapis.com/auth/devstorage.read_only",
			"https://www.googleapis.com/auth/logging.write",
			"https://www.googleapis.com/auth/monitoring"
		]
	}
}`, acctest.RandString(10), acctest.RandString(10))

var testAccContainerNodePool_withNodeConfigScopeAlias = fmt.Sprintf(`
resource "google_container_cluster" "cluster" {
	name = "tf-cluster-nodepool-test-%s"
	zone = "us-central1-a"
	initial_node_count = 1

	master_auth {
		username = "mr.yoda"
		password = "adoy.rm"
	}
}

resource "google_container_node_pool" "np_with_node_config_scope_alias" {
	name = "tf-nodepool-test-%s"
	zone = "us-central1-a"
	cluster = "${google_container_cluster.cluster.name}"
	initial_node_count = 1

	node_config {
		machine_type = "g1-small"
		disk_size_gb = 10
		oauth_scopes = ["compute-rw", "storage-ro", "logging-write", "monitoring"]
	}
}`, acctest.RandString(10), acctest.RandString(10))

func nodepoolCheckMatch(attributes map[string]string, attr string, gcp interface{}) string {
	if gcpList, ok := gcp.([]string); ok {
		return nodepoolCheckListMatch(attributes, attr, gcpList)
	}
	tf := attributes[attr]
	if tf != gcp {
		return nodepoolMatchError(attr, tf, gcp)
	}
	return ""
}

func nodepoolCheckListMatch(attributes map[string]string, attr string, gcpList []string) string {
	num, err := strconv.Atoi(attributes[attr+".#"])
	if err != nil {
		return fmt.Sprintf("Error in number conversion for attribute %s: %s", attr, err)
	}
	if num != len(gcpList) {
		return fmt.Sprintf("NodePool has mismatched %s size.\nTF Size: %d\nGCP Size: %d", attr, num, len(gcpList))
	}

	for i, gcp := range gcpList {
		if tf := attributes[fmt.Sprintf("%s.%d", attr, i)]; tf != gcp {
			return nodepoolMatchError(fmt.Sprintf("%s[%d]", attr, i), tf, gcp)
		}
	}

	return ""
}

func nodepoolMatchError(attr, tf string, gcp interface{}) string {
	return fmt.Sprintf("NodePool has mismatched %s.\nTF State: %+v\nGCP State: %+v", attr, tf, gcp)
}
