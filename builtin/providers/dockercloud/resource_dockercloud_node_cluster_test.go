package dockercloud

import (
	"fmt"
	"log"
	"reflect"
	"testing"

	"github.com/docker/go-dockercloud/dockercloud"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccCheckDockercloudNodeCluster_Basic(t *testing.T) {
	var nodeCluster dockercloud.NodeCluster

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDockercloudNodeClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDockercloudNodeClusterConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDockercloudNodeClusterExists("dockercloud_node_cluster.foobar", &nodeCluster),
					testAccCheckDockercloudNodeClusterAttributes(&nodeCluster),
					resource.TestCheckResourceAttr(
						"dockercloud_node_cluster.foobar", "name", "foobar-test-terraform"),
					resource.TestCheckResourceAttr(
						"dockercloud_node_cluster.foobar", "node_provider", "aws"),
					resource.TestCheckResourceAttr(
						"dockercloud_node_cluster.foobar", "size", "t2.nano"),
					resource.TestCheckResourceAttr(
						"dockercloud_node_cluster.foobar", "region", "us-east-1"),
				),
			},
		},
	})
}

func testAccCheckDockercloudNodeClusterDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "dockercloud_node_cluster" {
			continue
		}

		nodeCluster, err := dockercloud.GetNodeCluster(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error retrieving Node Cluster: %s", err)
		}

		if nodeCluster.State != "Terminated" {
			return fmt.Errorf("Node Cluster still running")
		}
	}

	return nil
}

func testAccCheckDockercloudNodeClusterExists(n string, nodeCluster *dockercloud.NodeCluster) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		retrieveNodeCluster, err := dockercloud.GetNodeCluster(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error retrieving Node Cluster: %s", err)
		}

		if retrieveNodeCluster.Uuid != rs.Primary.ID {
			return fmt.Errorf("Node Cluster not found")
		}

		*nodeCluster = retrieveNodeCluster

		return nil
	}
}

func testAccCheckDockercloudNodeClusterAttributes(nodeCluster *dockercloud.NodeCluster) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if nodeCluster.Name != "foobar-test-terraform" {
			return fmt.Errorf("Bad name: %s", nodeCluster.Name)
		}

		if nodeCluster.Region != "/api/infra/v1/region/aws/us-east-1/" {
			return fmt.Errorf("Bad region: %s", nodeCluster.Region)
		}

		if nodeCluster.NodeType != "/api/infra/v1/nodetype/aws/t2.nano/" {
			return fmt.Errorf("Bad nodetype: %s", nodeCluster.NodeType)
		}

		if nodeCluster.Target_num_nodes != 1 || nodeCluster.Current_num_nodes != 1 {
			return fmt.Errorf(
				"Bad num_nodes: %d (current), %d (target)",
				nodeCluster.Current_num_nodes, nodeCluster.Target_num_nodes,
			)
		}

		return nil
	}
}

const testAccCheckDockercloudNodeClusterConfig_basic = `
resource "dockercloud_node_cluster" "foobar" {
    name = "foobar-test-terraform"
    node_provider = "aws"
    size = "t2.nano"
    region = "us-east-1"
}`

func TestFlattenNodeTags(t *testing.T) {
	expected := []string{"foo", "bar", "baz"}
	tags := flattenNodeTags([]dockercloud.NodeTag{
		dockercloud.NodeTag{Name: "foo"},
		dockercloud.NodeTag{Name: "bar"},
		dockercloud.NodeTag{Name: "baz"},
	})

	if !reflect.DeepEqual(tags, expected) {
		t.Fatalf("Node tags do not match, got: %s", tags)
	}
}

func TestParseResourceURI(t *testing.T) {
	provider, region, err := parseResourceURI("/api/infra/v1/region/aws/us-east-1/", regionBasePath)
	if err != nil {
		log.Fatal(err)
	}

	if provider != "aws" {
		log.Fatalf("Expected provider to be aws, got: %s", provider)
	}

	if region != "us-east-1" {
		log.Fatalf("Expected region to be us-east-1, got: %s", region)
	}

	provider, size, err := parseResourceURI("/api/infra/v1/nodetype/aws/t2.nano/", nodeTypeBasePath)
	if err != nil {
		log.Fatal(err)
	}

	if provider != "aws" {
		log.Fatalf("Expected provider to be aws, got: %s", provider)
	}

	if size != "t2.nano" {
		log.Fatalf("Expected region to be us-east-1, got: %s", size)
	}
}
